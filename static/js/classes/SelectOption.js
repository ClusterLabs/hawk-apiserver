/* select options are always generic (hot hardcoded)
 * In the constructor we create an option document.createElement("option")
 * Unlike the Select class, where we apply it to an already existing html select */
class SelectOption {
    #option;
    #defaultValue;
    #shortdesc;
    #longdesc;
    #type;
    /* #possibleValues for an SelectOption is confusing. SelectOption is already a possible value!
       Yes! However this possible values may have SUB-OPTIONS.
       For examlple, there is a Meta Attributes SELECT which has options (grep: rscDefaults):
        [ allow-migrate, is-managed, maintenance, migration-threshold, priority, multiple-active,
		  failure-timeout, resource-stickiness, target-role, restart-type, description,
		  requires, remote-node, remote-port, remote-addr, remote-connect-timeout ]
        And for example you want to specify target-role. This will create a NEW SELECT
        with it's possible values [Started, Stopped, Master].
        ( Those options are binary (yes/no) and there is no need in deeper sub-options,
         however, in future, recursive options might be a better design ).
    */
    #possibleValues;
    #required;

    constructor(name, type, defaultValue, shortdesc, longdesc, possibleValues) {
        this.#option = document.createElement("option");
        this.#option.textContent = name;
        this.#option.value = name; // it's just the textContent

        this.#type = type;
        this.#defaultValue = defaultValue;
        this.#shortdesc = shortdesc;
        this.#longdesc = longdesc;
        this.#possibleValues = possibleValues;
    }

    hide() {
        this.#option.selected = false;
        this.#option.hidden = true;
    }

    show() { this.#option.hidden = false; }

    select() { this.#option.selected = true; }

    unselect() { this.#option.selected = false; }

    getName() { return this.#option.textContent || ""; }
    getDefaultValue() { return this.#defaultValue || ""; }
    getShortDesc() { return this.#shortdesc || ""; }
    getLongDesc() { return this.#longdesc || ""; }
    getType() { return this.#type || ""; }
    getPossibleValues() { return this.#possibleValues || []; }
    getRequired() { return this.#required || ""; }

    getHTML() { return this.#option; }

    // sugar function for the Select.size to correctly count available options
    countMe() {
        return this.#option.value !== "" && !this.#option.hidden;
    }

}

// Expose globally
window.SelectOption = SelectOption;
