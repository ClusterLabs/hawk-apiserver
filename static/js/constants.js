const CLUSTER_STATUS_UNCLEAN = "unclean";
const CLUSTER_STATUS_ONLINE = "online";
const CLUSTER_STATUS_NOQUORUM = "noquorum";
const CLUSTER_STATUS_NOFENCING = "nofencing";
const CLUSTER_STATUS_OFFLINE = "offline";
const CLUSTER_STATUS_MAINTENANCE = "maintenance";

const CLONE_PROMOTED = "master"; // cib.xml uses the old syntax for master/slave
const CLONE_DEMOTED = "slave";

const RESOURCE_STATUS_STARTED = "started"
const RESOURCE_STATUS_STOPPED = "stopped"
const RESOURCE_STATUS_PROMOTED = "promoted"
const RESOURCE_STATUS_UNPROMOTED = "unpromoted"
const RESOURCE_STATUS_OFFLINE = "offline" // node is offline
const RESOURCE_STATUS_DEFAULT = ""
