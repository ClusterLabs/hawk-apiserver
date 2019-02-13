package main

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

//go:generate bash gen.sh
// Common function for pretty print.
// Give pretty print by default;
// Give nomal print for efficiency reason,
// by setting request header "PrettyPrint" as non "1" value on client.
func MarshalOut(r *http.Request, easyStruct interface{}) ([]byte, error) {
	value := r.Header.Get("PrettyPrint")
	if value == "" || value == "1" {
		return json.MarshalIndent(easyStruct, "", "  ")
	}
	return json.Marshal(easyStruct)
}

// Apis under api/v1/configuration
func handleConfiguration(w http.ResponseWriter, r *http.Request, cib_data string) bool {
	var cib Cib
	err := xml.Unmarshal([]byte(cib_data), &cib)
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

func handleStatusApi(w http.ResponseWriter, r *http.Request, mon_data string) bool {
	// parse xml into Cib struct
	var crmMon CrmMon
	err := xml.Unmarshal([]byte(mon_data), &crmMon)
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

func IsString(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.String:
		return true
	}
	return false
}

func IsPtr(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Ptr:
		return true
	}
	return false
}

func IsStruct(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Struct:
		return true
	}
	return false
}

func IsSlice(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Slice:
		return true
	}
	return false
}

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

func GetNumField(in interface{}) int {
	rv := reflect.ValueOf(in)
	if IsBlank(rv) {
		return 0
	} else if IsStruct(rv) {
		return reflect.TypeOf(in).NumField()
	} else if IsPtr(rv) {
		return reflect.Indirect(rv).NumField()
	} else if IsSlice(rv) {
		return rv.Len()
	}
	return 0
}

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
			child_tag := strings.Split(rt.Field(i).Tag.Get("xml"), ",")
			if !IsBlank(rv.Field(i)) {
				FetchContent(ch, -1, rv.Field(i).Interface(), child_tag[0], head)
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

func FetchNv2(in interface{}) map[string]interface{} {
	if GetNumField(in) == 0 {
		return nil
	}

	ch := make(chan string)
	nv := make(map[string]interface{})
	sub_nv := make(map[string]interface{})
	sub_slice := make([]map[string]interface{}, 0)
	sub_key := ""

	go FetchContent(ch, GetNumField(in), in)
	for n := range ch {
		res := strings.Split(n, ";")
		if res[0] == "@slice@" {
			sub_nv = make(map[string]interface{})
			sub_key = res[1]
			continue
		}
		if sub_key != "" {
			sub_nv[res[0]] = res[1]
		} else {
			nv[res[0]] = res[1]
		}
	}

	if sub_key != "" {
		sub_slice = append(sub_slice, sub_nv)
		nv[sub_key] = sub_slice
	}

	return nv
}

func FetchNv(in interface{}) map[string]string {
	if GetNumField(in) == 0 {
		return nil
	}

	ch := make(chan string)
	key_slice := make([]string, 0)
	value_slice := make([]string, 0)
	nv := make(map[string]string)

	go FetchContent(ch, GetNumField(in), in)
	for n := range ch {
		if n == "@slice@" {
			continue
		}
		res := strings.Split(n, ";")
		if res[0] == "name" {
			key_slice = append(key_slice, res[1])
		}
		if res[0] == "value" {
			value_slice = append(value_slice, res[1])
		}

	}

	for index, _ := range key_slice {
		nv[key_slice[index]] = value_slice[index]
	}
	return nv
}
