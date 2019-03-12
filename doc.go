// Copyright 2019 Kristoffer Grönlund <kgronlund@suse.com>
//
// HTTPS API server / minimalist web proxy for Hawk.
//
// This project currently provides a minimalistic web server which
// handles SSL certificate termination, proxying and static file serving
// for Hawk, a HA cluster dashboard and control interface written in Ruby
// on Rails.
//
// The primary goal for this project is to provide the minimal web server
// needed by Hawk while consuming as few system resources as
// possible. Second, it provides the `/monitor` API endpoint which
// handles long-lived connections from the frontend to enable instant
// updates of the interface on cluster events.
package main
