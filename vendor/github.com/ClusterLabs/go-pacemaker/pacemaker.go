// The pacemaker package provides an API for reading the Pacemaker cluster configuration (CIB).
// Copyright (C) 2017 Kristoffer Gronlund <kgronlund@suse.com>
// See LICENSE for license.
package pacemaker

import (
	"unsafe"
	"fmt"
	"strings"
	"runtime"
)

/*
#cgo pkg-config: libxml-2.0 glib-2.0 libqb pacemaker pacemaker-cib
#include <crm/cib.h>
#include <crm/services.h>
#include <crm/common/util.h>
#include <crm/common/xml.h>
#include <crm/common/mainloop.h>

// Flags returned by go_cib_register_notify_callbacks
// indicating which notifications were actually
// available to register (different connection types
// enable different sets of notifications)
#define GO_CIB_NOTIFY_DESTROY 0x1
#define GO_CIB_NOTIFY_ADDREMOVE 0x2

extern int go_cib_signon(cib_t* cib, const char* name, enum cib_conn_type type);
extern int go_cib_signoff(cib_t* cib);
extern int go_cib_query(cib_t * cib, const char *section, xmlNode ** output_data, int call_options);
extern unsigned int go_cib_register_notify_callbacks(cib_t * cib);
extern void go_add_idle_scheduler(GMainLoop* loop);
*/
import "C"

// Error type returned by the functions in this package.
type CibError struct {
	msg string
}

func (e *CibError) Error() string {
	return e.msg
}

// Internal function used to create a CibError instance
// from a pacemaker return code.
func formatErrorRc(rc int) *CibError {
	errorname := C.pcmk_errorname(C.int(rc))
	strerror := C.pcmk_strerror(C.int(rc))
	if errorname == nil {
		errorname = C.CString("")
		defer C.free(unsafe.Pointer(errorname))
	}
	if strerror == nil {
		strerror = C.CString("")
		defer C.free(unsafe.Pointer(strerror))
	}
	return &CibError{fmt.Sprintf("%d: %s %s", rc, C.GoString(errorname), C.GoString(strerror))}
}

// When connecting to Pacemaker, we have
// to declare which type of connection to
// use. Since the API is read-only at the
// moment, it only really makes sense to
// pass Query to functions that take a
// CibConnection parameter.
type CibConnection int

const (
	Query CibConnection = C.cib_query
	Command CibConnection = C.cib_command
	NoConnection CibConnection = C.cib_no_connection
	CommandNonBlocking CibConnection = C.cib_command_nonblocking
)

type CibOpenConfig struct {
	connection CibConnection
	file string
	shadow string
	server string
	user string
	passwd string
	port int
	encrypted bool
}

func ForQuery(config *CibOpenConfig) {
	config.connection = Query
}

func ForCommand(config *CibOpenConfig) {
	config.connection = Command
}

func ForNoConnection(config *CibOpenConfig) {
	config.connection = NoConnection
}

func ForCommandNonBlocking(config *CibOpenConfig) {
	config.connection = CommandNonBlocking
}

func FromFile(file string) func(*CibOpenConfig) {
	return func(config *CibOpenConfig) {
		config.file = file
	}
}

func FromShadow(shadow string) func(*CibOpenConfig) {
	return func(config *CibOpenConfig) {
		config.shadow = shadow
	}
}

func FromRemote(server, user, passwd string, port int, encrypted bool) func (*CibOpenConfig) {
	return func(config *CibOpenConfig) {
		config.server = server
		config.user = user
		config.passwd = passwd
		config.port = port
		config.encrypted = encrypted
	}
}

type Element struct {
	Type string
	Id string
	Attr map[string]string
	Elements []*Element
}

type CibEvent int

const (
	UpdateEvent CibEvent = 0
	DestroyEvent CibEvent = 1
)

//go:generate stringer -type=CibEvent

type CibEventFunc func(event CibEvent, doc *CibDocument)

type subscriptionData struct {
	Id int
	Callback CibEventFunc
}


// Root entity representing the CIB. Can be
// populated with CIB data if the Decode
// method is used.
type Cib struct {
	cCib *C.cib_t
	subscribers map[int]CibEventFunc
	notifications uint
}

type CibVersion struct {
	AdminEpoch int32
	Epoch int32
	NumUpdates int32
}

type CibDocument struct {
	xml *C.xmlNode
}

func (ver *CibVersion) String() string {
	return fmt.Sprintf("%d:%d:%d", ver.AdminEpoch, ver.Epoch, ver.NumUpdates)
}

func OpenCib(options ...func (*CibOpenConfig)) (*Cib, error) {
	var cib Cib
	config := CibOpenConfig{}
	for _, opt := range options {
		opt(&config)
	}
	if config.connection != Query && config.connection != Command {
		config.connection = Query
	}
	if config.file != "" {
		s := C.CString(config.file)
		defer C.free(unsafe.Pointer(s))
		cib.cCib = C.cib_file_new(s)
	} else if config.shadow != "" {
		s := C.CString(config.shadow)
		defer C.free(unsafe.Pointer(s))
		cib.cCib = C.cib_shadow_new(s)
	} else if config.server != "" {
		s := C.CString(config.server)
		u := C.CString(config.user)
		p := C.CString(config.passwd)
		defer C.free(unsafe.Pointer(s))
		defer C.free(unsafe.Pointer(u))
		defer C.free(unsafe.Pointer(p))
		var e C.int = 0
		if config.encrypted {
			e = 1
		}
		cib.cCib = C.cib_remote_new(s, u, p, (C.int)(config.port), (C.gboolean)(e))
	} else {
		cib.cCib = C.cib_new()
	}

	rc := C.go_cib_signon(cib.cCib, C.crm_system_name, (uint32)(config.connection))
	if rc != C.pcmk_ok {
		return nil, formatErrorRc((int)(rc))
	}

	return &cib, nil
}

func GetShadowFile(name string) string {
	s := C.CString(name)
	defer C.free(unsafe.Pointer(s))
	return C.GoString(C.get_shadow_file(s))
}

func (cib *Cib) Close() error {
	rc := C.go_cib_signoff(cib.cCib)
	if rc != C.pcmk_ok {
		return formatErrorRc((int)(rc))
	}
	C.cib_delete(cib.cCib)
	cib.cCib = nil
	return nil
}

func (doc *CibDocument) Version() *CibVersion {
	var admin_epoch C.int
	var epoch C.int
	var num_updates C.int
	ok := C.cib_version_details(doc.xml, (*C.int)(unsafe.Pointer(&admin_epoch)), (*C.int)(unsafe.Pointer(&epoch)), (*C.int)(unsafe.Pointer(&num_updates)))
	if ok == 1 {
		return &CibVersion{(int32)(admin_epoch), (int32)(epoch), (int32)(num_updates)}
	}
	return nil
}

func (doc *CibDocument) ToString() string {
	buffer := C.dump_xml_unformatted(doc.xml)
	defer C.free(unsafe.Pointer(buffer))
	return C.GoString(buffer)
}

func (doc *CibDocument) Close() {
	C.free_xml(doc.xml)
}


func (cib *Cib) queryImpl(xpath string, nochildren bool) (*C.xmlNode, error) {
	var root *C.xmlNode
	var rc C.int

	var opts C.int

	opts = C.cib_sync_call + C.cib_scope_local

	if xpath != "" {
		opts += C.cib_xpath
	}

	if nochildren {
		opts += C.cib_no_children
	}

	if xpath != "" {
		xp := C.CString(xpath)
		defer C.free(unsafe.Pointer(xp))
		rc = C.go_cib_query(cib.cCib, xp, (**C.xmlNode)(unsafe.Pointer(&root)), opts)
	} else {
		rc = C.go_cib_query(cib.cCib, nil, (**C.xmlNode)(unsafe.Pointer(&root)), opts)
	}
	if rc != C.pcmk_ok {
		return nil, formatErrorRc((int)(rc))
	}
	return root, nil
}

func (cib *Cib) Version() (*CibVersion, error) {
	var admin_epoch C.int
	var epoch C.int
	var num_updates C.int

	root, err := cib.queryImpl("/cib", true)
	if err != nil {
		return nil, err
	}
	defer C.free_xml(root)
	ok := C.cib_version_details(root, (*C.int)(unsafe.Pointer(&admin_epoch)), (*C.int)(unsafe.Pointer(&epoch)), (*C.int)(unsafe.Pointer(&num_updates)))
	if ok == 1 {
		return &CibVersion{(int32)(admin_epoch), (int32)(epoch), (int32)(num_updates)}, nil
	}
	return nil, &CibError{"Failed to get CIB version details"}
}

func (cib *Cib) Query() (*CibDocument, error) {
	var root *C.xmlNode
	root, err := cib.queryImpl("", false)
	if err != nil {
		return nil, err
	}

	return &CibDocument{root}, nil
}

func (cib *Cib) QueryNoChildren() (*CibDocument, error) {
	var root *C.xmlNode
	root, err := cib.queryImpl("", true)
	if err != nil {
		return nil, err
	}
	return &CibDocument{root}, nil
}


func (cib *Cib) QueryXPath(xpath string) (*CibDocument, error) {
	var root *C.xmlNode
	root, err := cib.queryImpl(xpath, false)
	if err != nil {
		return nil, err
	}
	return &CibDocument{root}, nil
}

func (cib *Cib) QueryXPathNoChildren(xpath string) (*CibDocument, error) {
	var root *C.xmlNode
	root, err := cib.queryImpl(xpath, true)
	if err != nil {
		return nil, err
	}
	return &CibDocument{root}, nil
}

func init() {
	s := C.CString("go-pacemaker")
	C.crm_log_init(s, C.LOG_CRIT, 0, 0, 0, nil, 1)
	C.free(unsafe.Pointer(s))
}

func IsTrue(bstr string) bool {
	sl := strings.ToLower(bstr)
	return sl == "true" || sl == "on" || sl == "yes" || sl == "y" || sl == "1"
}

var the_cib *Cib

func (cib *Cib) Subscribers() map[int]CibEventFunc {
	return cib.subscribers
}

func (cib *Cib) Subscribe(callback CibEventFunc) (uint, error) {
	the_cib = cib
	if cib.subscribers == nil {
		cib.subscribers = make(map[int]CibEventFunc)
		flags := C.go_cib_register_notify_callbacks(cib.cCib)
		cib.notifications = uint(flags)
	}
	id := len(cib.subscribers)
	cib.subscribers[id] = callback
	return cib.notifications, nil
}

//export diffNotifyCallback
func diffNotifyCallback(current_cib *C.xmlNode) {
	for _, callback := range the_cib.subscribers {
		callback(UpdateEvent, &CibDocument{current_cib})
	}
}

//export destroyNotifyCallback
func destroyNotifyCallback() {
	for _, callback := range the_cib.subscribers {
		callback(DestroyEvent, nil)
	}
}

//export goMainloopSched
func goMainloopSched() {
	runtime.Gosched()
}

func Mainloop() {
	mainloop := C.g_main_loop_new(nil, C.FALSE)
	C.go_add_idle_scheduler(mainloop)
	C.g_main_loop_run(mainloop)
	C.g_main_loop_unref(mainloop)
}
