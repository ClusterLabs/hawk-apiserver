package api

//go:generate bash gen.sh

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"reflect"
	"strings"
)

// MarshalOut is used to pretty-print JSON data if the HTTP header
// "PrettyPrint" is set to "1" or is not set, so to disable
// pretty-printing, set any other value in the header.
func MarshalOut(r *http.Request, easyStruct interface{}) ([]byte, error) {
	value := r.Header.Get("PrettyPrint")
	if value == "" || value == "1" {
		return json.MarshalIndent(easyStruct, "", "  ")
	}
	return json.Marshal(easyStruct)
}

// HandleConfiguration handles APIs under api/v1/configuration
func HandleConfiguration(w http.ResponseWriter, r *http.Request, cibData string) bool {
	var cib Cib
	err := xml.Unmarshal([]byte(cibData), &cib)
	if err != nil {
		log.Error(err)
		return false
	}

	urllist := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	w.Header().Set("Content-Type", "application/json")

	// mapping between url type and struct casting function
	configHandle := map[string]func([]string, *Cib) (bool, interface{}){
		"nodes":            handleConfigNodes,
		"resources":        handleConfigResources,
		"primitives":       handleConfigPrimitive,
		"groups":           handleConfigGroup,
		"masters":          handleConfigMaster,
		"clones":           handleConfigClone,
		"bundles":          handleConfigBundle,
		"cluster_property": handleConfigCluster,
		"constraints":      handleConfigConstraints,
		"locations":        handleConfigLocation,
		"colocations":      handleConfigColocation,
		"orders":           handleConfigOrder,
		"rsc_defaults":     handleConfigRscDefaults,
		"op_defaults":      handleConfigOpDefaults,
		"alerts":           handleConfigAlerts,
		"tags":             handleConfigTags,
		"acls":             handleConfigAcls,
		"fencing":          handleConfigFencing,
	}

	// return simple and easy understand struct
	rc, easyStruct := configHandle[urllist[3]](urllist, &cib)
	if !rc {
		http.Error(w, fmt.Sprintf("No route for %v.", r.URL.Path), 500)
		return false
	}

	jsonData, jsonError := MarshalOut(r, easyStruct)
	if jsonError != nil {
		log.Error(jsonError)
		return false
	}

	io.WriteString(w, string(jsonData)+"\n")
	return true
}

// HandleStatus handles status API requests.
func HandleStatus(w http.ResponseWriter, r *http.Request, monData string) bool {
	// parse xml into Cib struct
	var crmMon CrmMon
	err := xml.Unmarshal([]byte(monData), &crmMon)
	if err != nil {
		log.Error(err)
		return false
	}

	urllist := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	w.Header().Set("Content-Type", "application/json")

	configHandle := map[string]func([]string, *CrmMon) (bool, interface{}){
		"nodes":     handleStateNodes,
		"resources": handleStateResources,
		"summary":   handleStateSummary,
		"failures":  handleStateFailures,
	}
	rc, easyStruct := configHandle[urllist[3]](urllist, &crmMon)
	if !rc {
		http.Error(w, fmt.Sprintf("No route for %v.", r.URL.Path), 500)
		return false
	}

	jsonData, jsonError := MarshalOut(r, easyStruct)
	if jsonError != nil {
		log.Error(jsonError)
		return false
	}

	io.WriteString(w, string(jsonData)+"\n")
	return true
}

// IsString checks if value is a string.
func IsString(value reflect.Value) bool {
	return value.Kind() == reflect.String
}

// IsPtr checks if value is a bool.
func IsPtr(value reflect.Value) bool {
	return value.Kind() == reflect.Ptr
}

// IsStruct checks if value is a struct.
func IsStruct(value reflect.Value) bool {
	return value.Kind() == reflect.Struct
}

// IsMap checks if value is a map.
func IsMap(value reflect.Value) bool {
	return value.Kind() == reflect.Map
}

// IsSlice checks if value is a slice.
func IsSlice(value reflect.Value) bool {
	return value.Kind() == reflect.Slice
}

// IsBlank returns true if value is unset (empty string, zero value, etc.).
func IsBlank(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.String:
		return value.Len() == 0
	case reflect.Bool:
		return !value.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return value.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return value.IsNil()
	}
	return reflect.DeepEqual(value.Interface(), reflect.Zero(value.Type()).Interface())
}

func retryGetNumField(rv reflect.Value) int {
	if IsBlank(rv) {
		return 0
	} else if IsStruct(rv) {
		return rv.NumField()
	} else if IsPtr(rv) {
		return retryGetNumField(reflect.Indirect(rv))
	} else if IsSlice(rv) {
		return rv.Len()
	}
	return 0
}

// GetNumField returns NumField() for structs or pointers to structs, and len for slices
func GetNumField(in interface{}) int {
	return retryGetNumField(reflect.ValueOf(in))
}

// FetchContent parses the content of an XML structure and
// sends anything interesting into the provided channel.
//
// For some specific structs from api_structs.go,
// like ClusterPropertySet, Operations.Op or MetaAttributes,
// it's more flexible using reflect recurrently to parse the contents and
// extracting useful data in them, instead of defining fixed struct
// Parameters:
// ch			channel to send data
// outerFieldsNum	flags to close the channel
// in			data, child and parent's xml tag
func FetchContent(ch chan string, outerFieldsNum int, in ...interface{}) {
	data := in[0]
	// key is child xml tag
	// header is parent xml tag
	var key, header interface{}
	if len(in) == 2 {
		key = in[1]
	} else if len(in) == 3 {
		key, header = in[1], in[2]
	}

	rv := reflect.ValueOf(data)
	switch rv.Kind() {
	case reflect.Ptr:
		FetchContent(ch, outerFieldsNum, reflect.Indirect(rv).Interface(), key, header)

	case reflect.Struct:
		rt := reflect.TypeOf(data)
		head := rt.Field(0).Tag.Get("xml")

		for i := 1; i < rt.NumField(); i++ {
			childTag := strings.Split(rt.Field(i).Tag.Get("xml"), ",")
			if !IsBlank(rv.Field(i)) {
				FetchContent(ch, -1, rv.Field(i).Interface(), childTag[0], head)
			}

			if i == outerFieldsNum-1 {
				close(ch)
			}
		}

	case reflect.Slice:
		var sendmsg string
		if key != nil {
			sendmsg = "@slice@" + ";" + header.(string)
		} else {
			sendmsg = "@slice@"
		}
		for i := 0; i < rv.Len(); i++ {
			ch <- sendmsg
			FetchContent(ch, -1, rv.Index(i).Interface(), key, header)

			if i == outerFieldsNum-1 {
				close(ch)
			}
		}

	case reflect.String:
		ch <- fmt.Sprintf("%s;%s;%s", key.(string), data.(string), header.(string))
	}
}

// FetchNV2 scans the untyped input for key/value pairs.
func FetchNV2(in interface{}) map[string]interface{} {
	if GetNumField(in) == 0 {
		return nil
	}

	ch := make(chan string)
	nv := make(map[string]interface{})
	subNV := make(map[string]interface{})
	subSlice := make([]map[string]interface{}, 0)
	subKey := ""

	go FetchContent(ch, GetNumField(in), in)
	for n := range ch {
		res := strings.Split(n, ";")
		if res[0] == "@slice@" {
			subNV = make(map[string]interface{})
			subKey = res[1]
			continue
		}
		if subKey != "" {
			subNV[res[0]] = res[1]
		} else {
			nv[res[0]] = res[1]
		}
	}

	if subKey != "" {
		subSlice = append(subSlice, subNV)
		nv[subKey] = subSlice
	}

	return nv
}

// FetchNV tries to parse name/value pairs
// from the provided object.
func FetchNV(in interface{}) map[string]string {
	if GetNumField(in) == 0 {
		return nil
	}

	ch := make(chan string)
	keySlice := make([]string, 0)
	valueSlice := make([]string, 0)
	nv := make(map[string]string)

	go FetchContent(ch, GetNumField(in), in)
	for n := range ch {
		if n == "@slice@" {
			continue
		}
		res := strings.Split(n, ";")
		if res[0] == "name" {
			keySlice = append(keySlice, res[1])
		}
		if res[0] == "value" {
			valueSlice = append(valueSlice, res[1])
		}

	}

	for index, key := range keySlice {
		nv[key] = valueSlice[index]
	}
	return nv
}
