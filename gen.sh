venv_dir="hawk_api_gen_go_struct_from_cibschema"

rm -rf "$venv_dir"
/usr/bin/env python3 -m venv "$venv_dir"
source "$venv_dir"/bin/activate
pip install CibToGoStruct &> /dev/null
cibToGoStruct
rm -rf "$venv_dir"
