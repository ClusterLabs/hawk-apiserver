// The pacemaker package provides an API for reading the Pacemaker cluster configuration (CIB).
// Copyright (C) 2017 Kristoffer Gronlund <kgronlund@suse.com>
// See LICENSE for license.
package pacemaker

/*

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


#define F_CIB_UPDATE_RESULT "cib_update_result"

static cib_t *s_cib = NULL;

int go_cib_signon(cib_t* cib, const char* name, enum cib_conn_type type) {
	int rc;
	rc = cib->cmds->signon(cib, name, type);
	return rc;
}

int go_cib_signoff(cib_t* cib) {
	int rc;
	rc = cib->cmds->signoff(cib);
	return rc;
}

int go_cib_query(cib_t * cib, const char *section, xmlNode ** output_data, int call_options) {
	int rc;
	rc = cib->cmds->query(cib, section, output_data, call_options);
	return rc;
}

static void go_cib_destroy_cb(gpointer user_data) {
	extern void destroyNotifyCallback();
	destroyNotifyCallback();
}

static void go_cib_notify_cb(const char *event, xmlNode * msg) {
	int rc;
	rc = pcmk_ok;

	xmlNode *current_cib;
	xmlNode *diff = get_message_xml(msg, F_CIB_UPDATE_RESULT);

	s_cib->cmds->query(s_cib, NULL, &current_cib, cib_scope_local | cib_sync_call);

	extern void diffNotifyCallback(xmlNode*);
	diffNotifyCallback(current_cib);

	free_xml(current_cib);
}


unsigned int go_cib_register_notify_callbacks(cib_t * cib) {
	int rc;
	unsigned int flags;

	s_cib = cib;
	flags = 0;

	rc = cib->cmds->set_connection_dnotify(cib, go_cib_destroy_cb);
	if (rc == pcmk_ok) {
		flags |= GO_CIB_NOTIFY_DESTROY;
	}
	rc = cib->cmds->del_notify_callback(cib, T_CIB_DIFF_NOTIFY, go_cib_notify_cb);
	if (rc == pcmk_ok) {
		flags |= GO_CIB_NOTIFY_ADDREMOVE;
	}
	rc = cib->cmds->add_notify_callback(cib, T_CIB_DIFF_NOTIFY, go_cib_notify_cb);
	if (rc == pcmk_ok) {
		flags |= GO_CIB_NOTIFY_ADDREMOVE;
	}
	return flags;
}

static gboolean idle_callback(gpointer user_data) {
	extern void goMainloopSched();
	goMainloopSched();
}

void go_add_idle_scheduler(GMainLoop* loop) {
	g_idle_add(&idle_callback, loop);
}

*/
import "C"

