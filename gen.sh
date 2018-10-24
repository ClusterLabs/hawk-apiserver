
if which cibToGoStruct 1>/dev/null 2>&1; then
	cibToGoStruct
else
	venv_dir="hawk_api_gen_go_struct_from_cibschema"
	[ -d "$venv_dir" ] && rm -rf "$venv_dir"
	/usr/bin/env python3 -m venv "$venv_dir"
	source "$venv_dir"/bin/activate
	pip install CibToGoStruct &> /dev/null
	cibToGoStruct
	rm -rf "$venv_dir"
fi
[ -f api_structs.go ] && go fmt api_structs.go
