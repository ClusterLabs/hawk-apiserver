package api

func GetRscDefaults() []MetaParameter {
	// return a copy to prevent modification
	result := make([]MetaParameter, len(rscDefaults))
	copy(result, rscDefaults)
	return result
}

func GetCloneDefaults() []MetaParameter {
	// return a copy to prevent modification
	result := make([]MetaParameter, len(cloneDefaults))
	// TODO: cloneDefaults["clone-max"].Detault := number of nodes in cluster
	copy(result, cloneDefaults)
	return result
}

/*
func GetOpDefaults() []MetaParameter {
	// return a copy to prevent modification
	result := make([]MetaParameter, len(opDefaults))
	copy(result, opDefaults)
	return result
}
*/

func GetOpDescriptions() []MetaParameter {
	// return a copy to prevent modification
	result := make([]MetaParameter, len(opDescriptions))
	copy(result, opDescriptions)
	return result
}

/*****************************
 * default meta attributes
 *****************************/

// copied from hawk -> tableless.rb --> RSC_DEFAULTS
// TODO: consider using hash-map Name -> Longdesc,Content
var rscDefaults = []MetaParameter{
	{
		Name:     "allow-migrate",
		Longdesc: "Set to true if the resource agent supports the migrate action",
		Content: ContentAttr{
			Type:    "boolean",
			Default: "false",
		},
	},
	{
		Name:     "is-managed",
		Longdesc: "Is the cluster allowed to start and stop the resource?",
		Content: ContentAttr{
			Type:    "boolean",
			Default: "true",
		},
	},
	{
		Name:     "maintenance",
		Longdesc: "Resources in maintenance mode are not monitored by the cluster.",
		Content: ContentAttr{
			Type:    "boolean",
			Default: "false",
		},
	},
	{
		Name:     "interval-origin",
		Longdesc: "For a recurring action, schedule the action for this ISO 8601 time plus a multiple of the action's interval instead of immediately after the resource gains the monitored role.",
		Content: ContentAttr{
			Type:    "string",
			Default: "",
		},
	},
	{
		Name:     "migration-threshold",
		Longdesc: "How many failures may occur for this resource on a node before it's marked ineligible...",
		Content: ContentAttr{
			Type:    "integer",
			Default: "0",
		},
	},
	{
		Name:     "priority",
		Longdesc: "If not all resources can be active, lower priority ones will be stopped first.",
		Content: ContentAttr{
			Type:    "integer",
			Default: "0",
		},
	},
	{
		Name:     "multiple-active",
		Longdesc: "What should the cluster do if it finds the resource active on more than one node?",
		Content: ContentAttr{
			Type:           "enum",
			Default:        "stop_start",
			PossibleValues: []string{"block", "stop_only", "stop_start", "stop_unexpected"},
		},
	},
	{
		Name:     "failure-timeout",
		Longdesc: "Time to wait before considering the failure 'expired'.",
		Content: ContentAttr{
			Type:    "integer",
			Default: "0",
		},
	},
	{
		Name:     "resource-stickiness",
		Longdesc: "How much does the resource prefer to stay where it is?",
		Content: ContentAttr{
			Type:    "integer",
			Default: "0",
		},
	},
	{
		Name:     "target-role",
		Longdesc: "What state should the cluster try to maintain for this resource?",
		Content: ContentAttr{
			Type:           "enum",
			Default:        "Stopped",
			PossibleValues: []string{"Started", "Stopped", "Unpromoted", "Promoted"},
		},
	},
	{
		Name: "restart-type",
		Content: ContentAttr{
			Type:           "enum",
			Default:        "ignore",
			PossibleValues: []string{"ignore", "restart"},
		},
	},
	{
		Name: "description",
		Content: ContentAttr{
			Type:    "string",
			Default: "",
		},
	},
	{
		Name:     "requires",
		Longdesc: "Conditions required to start the resource.",
		Content: ContentAttr{
			Type:           "enum",
			Default:        "fencing",
			PossibleValues: []string{"nothing", "quorum", "fencing", "unfencing"},
		},
	},
	{
		Name:     "provides",
		Longdesc: "A special capability provided by a fencing resource. Currently, the only meaningful capability is unfencing.",
		Content: ContentAttr{
			Type:           "enum",
			Default:        "",
			PossibleValues: []string{"unfencing"},
		},
	},
	{
		Name:     "remote-node",
		Longdesc: "The name of the remote-node this resource defines.",
		Content: ContentAttr{
			Type:    "string",
			Default: "",
		},
	},
	{
		Name:     "remote-port",
		Longdesc: "Port used for the guest connection.",
		Content: ContentAttr{
			Type:    "integer",
			Default: "3121",
		},
	},
	{
		Name:     "remote-addr",
		Longdesc: "The IP address or hostname for remote-node connection.",
		Content: ContentAttr{
			Type:    "string",
			Default: "",
		},
	},
	{
		Name:     "remote-connect-timeout",
		Longdesc: "Timeout before a pending guest connection fails.",
		Content: ContentAttr{
			Type:    "string",
			Default: "60s",
		},
	},
	{
		Name:     "critical",
		Longdesc: "Use this value as the default for influence in colocation constraints involving this resource and in implicit colocation constraints created for groups.",
		Content: ContentAttr{
			Type:    "boolean",
			Default: "true",
		},
	},
	{
		Name:     "allow-unhealthy-nodes",
		Longdesc: "Whether the resource may run on a node even if the node's health score would otherwise prevent it.",
		Content: ContentAttr{
			Type:    "boolean",
			Default: "false",
		},
	},
	{
		Name:     "container-attribute-target",
		Longdesc: "Where to check user-defined node attributes. The value host selects the underlying physical host; any other value selects the local node.",
		Content: ContentAttr{
			Type:    "string",
			Default: "",
		},
	},
}

// copied from hawk -> clones.rb -> 'def mapping'
var cloneDefaults = []MetaParameter{
	{
		Name:     "is-managed",
		Longdesc: "Is the cluster allowed to start and stop the resource?",
		Content: ContentAttr{
			Type:    "boolean",
			Default: "false",
		},
	},
	{
		Name:     "maintenance",
		Longdesc: "Resources in maintenance mode are not monitored by the cluster.",
		Content: ContentAttr{
			Type:    "boolean",
			Default: "false",
		},
	},
	{
		Name:     "priority",
		Longdesc: "If not all resources can be active, the cluster will stop lower priority resources in order to keep higher priority ones active.",
		Content: ContentAttr{
			Type:    "integer",
			Default: "0",
		},
	},
	{
		Name:     "promotable",
		Longdesc: "Resource can be promoted (previously we would say: the resource can become a master).",
		Content: ContentAttr{
			Type:    "boolean",
			Default: "true",
		},
	},
	{
		Name:     "target-role",
		Longdesc: "What state should the cluster attempt to keep this resource in?",
		Content: ContentAttr{
			Type:           "enum",
			Default:        "Stopped",
			PossibleValues: []string{"Started", "Stopped", "Unpromoted", "Promoted"},
		},
	},
	{
		Name:     "clone-max",
		Longdesc: "How many copies of the resource to start. Defaults to the number of nodes in the cluster.",
		Content: ContentAttr{
			Type:    "integer",
			Default: "1", // TODO: should be the number of nodes in the cluster
		},
	},
	{
		Name:     "clone-node-max",
		Longdesc: "How many copies of the resource can be started on a single node. Defaults to 1.",
		Content: ContentAttr{
			Type:    "integer",
			Default: "1",
		},
	},
	{
		Name: "clone-state",
		Content: ContentAttr{
			Type:    "string",
			Default: "",
		},
	},
	{
		Name: "description",
		Content: ContentAttr{
			Type:    "string",
			Default: "",
		},
	},
	{
		Name:     "notify",
		Longdesc: "When stopping or starting a copy of the clone, tell all the other copies beforehand and when the action was successful.",
		Content: ContentAttr{
			Type:    "boolean",
			Default: "false",
		},
	},
	{
		Name:     "globally-unique",
		Longdesc: "Does each copy of the clone perform a different function?",
		Content: ContentAttr{
			Type:    "boolean",
			Default: "true",
		},
	},
	{
		Name:     "ordered",
		Longdesc: "Should the copies be started in series (instead of in parallel)?",
		Content: ContentAttr{
			Type:    "boolean",
			Default: "false",
		},
	},
	{
		Name:     "interleave",
		Longdesc: "Changes the behavior of ordering constraints (between clones/masters) so that instances can start/stop as soon as their peer instance has (rather than waiting for every instance of the other clone has).",
		Content: ContentAttr{
			Type:    "boolean",
			Default: "false",
		},
	},
}

// copied from hawk -> tableless.rb --> OP_DEFAULTS
// TODO: consider using hash-map Name -> Longdesc,Content
/*
var opDefaults = []MetaParameter{
	{
		Name:     "interval",
		Longdesc: "How frequently(in seconds) to perform the operation.",
		Content: ContentAttr{
			Type:     "string",
			Default:  "0",
			Required: "false",
		},
	},
	{
		Name:     "timeout",
		Longdesc: "How long to wait before declaring the action has failed.",
		Content: ContentAttr{
			Type:     "string",
			Default:  "20",
			Required: "true",
		},
	},
	{
		Name:     "requires",
		Longdesc: "What conditions need to be satisfied before this action occurs.",
		Content: ContentAttr{
			Type:           "enum",
			Default:        "fencing",
			PossibleValues: []string{"nothing", "quorum", "fencing"},
		},
	},
	{
		Name:     "enabled",
		Longdesc: "If false, the operation is treated as if it does not exist.",
		Content: ContentAttr{
			Type:    "boolean",
			Default: "true",
		},
	},
	{
		Name:     "role",
		Longdesc: "This option only makes sense for recurring operations. It restricts the operation to a specific role. The truly paranoid can even specify role=Stopped which allows the cluster to detect an admin that manually started cluster services.",
		Content: ContentAttr{
			Type:           "enum",
			Default:        "",
			PossibleValues: []string{"Stopped", "Started", "Unpromoted", "Promoted"},
		},
	},
	{
		Name:     "on-fail",
		Longdesc: "The action to take if this action ever fails.",
		Content: ContentAttr{
			Type:           "enum",
			Default:        "stop",
			PossibleValues: []string{"ignore", "block", "stop", "restart", "standby", "fence"},
		},
	},
	{
		Name:     "start-delay",
		Longdesc: "The delay time(in seconds) before doing the operation",
		Content: ContentAttr{
			Type:    "string",
			Default: "0",
		},
	},
	{
		Name:     "interval-origin",
		Longdesc: "The start time of action interval. Follow the ISO8601 standard.",
		Content: ContentAttr{
			Type:    "string",
			Default: "",
		},
	},
	{
		Name:     "record-pending",
		Longdesc: "If true, the intention to perform the operation is recorded so that GUIs and CLI tools can indicate that an operation is in progress.",
		Content: ContentAttr{
			Type:    "boolean",
			Default: "false",
		},
	},
	{
		Name: "description",
		Content: ContentAttr{
			Type:    "string",
			Default: "",
		},
	},
}
*/

// ActionDefaults provides named access to the defaults of an action.
type ActionDefaults struct {
	Depth          FullPrimitive_Action_MetaParameter
	Interval       FullPrimitive_Action_MetaParameter
	Timeout        FullPrimitive_Action_MetaParameter
	Requires       FullPrimitive_Action_MetaParameter
	Enabled        FullPrimitive_Action_MetaParameter
	Role           FullPrimitive_Action_MetaParameter
	OnFail         FullPrimitive_Action_MetaParameter
	StartDelay     FullPrimitive_Action_MetaParameter
	IntervalOrigin FullPrimitive_Action_MetaParameter
	RecordPending  FullPrimitive_Action_MetaParameter
	Description    FullPrimitive_Action_MetaParameter
}

var actionDefaults = ActionDefaults{
	Depth: FullPrimitive_Action_MetaParameter{
		Name: "depth",
	},
	Interval: FullPrimitive_Action_MetaParameter{
		Name:     "interval",
		Longdesc: "How frequently(in seconds) to perform the operation.",
		Content: FullPrimitive_Action_ContentAttr{
			Type:     "string",
			Default:  "0",
			Required: "false",
		},
	},
	Timeout: FullPrimitive_Action_MetaParameter{
		Name:     "timeout",
		Longdesc: "How long to wait before declaring the action has failed.",
		Content: FullPrimitive_Action_ContentAttr{
			Type:     "string",
			Default:  "20",
			Required: "true",
		},
	},
	Requires: FullPrimitive_Action_MetaParameter{
		Name:     "requires",
		Longdesc: "What conditions need to be satisfied before this action occurs.",
		Content: FullPrimitive_Action_ContentAttr{
			Type:           "enum",
			Default:        "fencing",
			PossibleValues: []string{"nothing", "quorum", "fencing"},
		},
	},
	Enabled: FullPrimitive_Action_MetaParameter{
		Name:     "enabled",
		Longdesc: "If false, the operation is treated as if it does not exist.",
		Content: FullPrimitive_Action_ContentAttr{
			Type:    "boolean",
			Default: "true",
		},
	},
	Role: FullPrimitive_Action_MetaParameter{
		Name:     "role",
		Longdesc: "This option only makes sense for recurring operations. It restricts the operation to a specific role. The truly paranoid can even specify role=Stopped which allows the cluster to detect an admin that manually started cluster services.",
		Content: FullPrimitive_Action_ContentAttr{
			Type:           "enum",
			Default:        "",
			PossibleValues: []string{"Stopped", "Started", "Unpromoted", "Promoted"},
		},
	},
	OnFail: FullPrimitive_Action_MetaParameter{
		Name:     "on-fail",
		Longdesc: "The action to take if this action ever fails.",
		Content: FullPrimitive_Action_ContentAttr{
			Type:           "enum",
			Default:        "stop",
			PossibleValues: []string{"ignore", "block", "stop", "restart", "standby", "fence"},
		},
	},
	StartDelay: FullPrimitive_Action_MetaParameter{
		Name:     "start-delay",
		Longdesc: "The delay time(in seconds) before doing the operation",
		Content: FullPrimitive_Action_ContentAttr{
			Type:    "string",
			Default: "0",
		},
	},
	IntervalOrigin: FullPrimitive_Action_MetaParameter{
		Name:     "interval-origin",
		Longdesc: "The start time of action interval. Follow the ISO8601 standard.",
		Content: FullPrimitive_Action_ContentAttr{
			Type:    "string",
			Default: "",
		},
	},
	RecordPending: FullPrimitive_Action_MetaParameter{
		Name:     "record-pending",
		Longdesc: "If true, the intention to perform the operation is recorded so that GUIs and CLI tools can indicate that an operation is in progress.",
		Content: FullPrimitive_Action_ContentAttr{
			Type:    "boolean",
			Default: "false",
		},
	},
	Description: FullPrimitive_Action_MetaParameter{
		Name: "description",
		Content: FullPrimitive_Action_ContentAttr{
			Type:    "string",
			Default: "",
		},
	},
}

// opDescriptions comes from hawk/app/models/template.rb
// TODO: consider using hash-map Name -> Description
var opDescriptions = []MetaParameter{{
	Name:      "template",
	Shortdesc: "Template",
	Longdesc:  "Resource template to inherit from.",
}, {
	Name:      "clazz",
	Shortdesc: "Template",
	Longdesc:  "Resource template to inherit from.",
}, {
	Name:      "provider",
	Shortdesc: "Provider",
	Longdesc:  "Vendor or project which provided the resource agent.",
}, {
	Name:      "type",
	Shortdesc: "Type",
	Longdesc:  "Resource agent name.",
}, {
	Name:      "op-start",
	Shortdesc: "Start",
	Longdesc:  "After the specified timeout period, the operation will be treated as failed.",
}, {
	Name:      "op-stop",
	Shortdesc: "Stop",
	Longdesc:  "After the specified timeout period, the operation will be treated as failed.",
}, {
	Name:      "op-monitor",
	Shortdesc: "Monitor",
	Longdesc:  "Define a monitor operation to instruct the cluster to ensure that the resource is still healthy.",
},
}
