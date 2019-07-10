#!/usr/bin/env python3

# this script was initially imported from https://github.com/liangxin1300/CibToGo.

import subprocess
import sys
import os
import re
from lxml import etree, objectify
from jinja2 import Environment


otherStructTemplate = """
type TypeIndex struct {
        Type  string
        Index int
}
"""


goStructTemplate = """
{%- macro struct_type(child) -%}
    {%- if child.type.startswith("point_") or child.type.startswith("slice_") %}
        *{{ convert_name('_'.join(child.type.split('_')[1:])) }}
    {%- elif child.type == "slice" %}
        []*{{ convert_name(child.name) }}
    {%- elif child.type in ("string", "int") %}
        {{ child.type }}
    {%- else %}
        *{{ convert_name(child.type) }}
    {%- endif -%}
{%- endmacro -%}

{%- macro struct_tag(child) -%}
    {%- if child.xmltag != "-" or child.jsontag != "-" %}
        `xml:"{{ child.xmltag }}" json:"{{ child.jsontag }}"`
    {%- endif -%}
{%- endmacro -%}

type {{ convert_name(node.name) }} struct {
    XMLNAME    xml.Name    `xml:"{{ node.name }}" json:"-"`
{% for child in node.children %}
    {{ convert_name(child.name) }}{{ struct_type(child) }}{{ struct_tag(child) }}
{% endfor %}
{% if node.name in ("configuration", "nodes", "resources", "constraints") %}
    URLType string    `json:"-"`
{% endif %}
{% if node.name in ("nodes", "resources", "constraints") %}
    URLIndex int    `json:"-"`
{% endif %}
}
"""


def convert_name(name):
    res = ""
    words = re.split(r'[-_\.]', name)
    for word in words:
        res += word.capitalize()
    return res


def node_exists(allnodes, node):
    for n in allnodes:
        if n.name == node.name:
            return True
    return False


class Node:
    '''
    stand for element
    '''
    def __init__(self, name=None):
        self.name = name
        self.children = []

    def __str__(self):
        res = "%s> " % self.name
        for item in self.children:
            res += "%s(%s %s;%s); " % (item.name, item.type, item.xmltag, item.jsontag)
        return res

    def append(self, child):
        for ch in self.children:
            if ch.name == child.name:
                return
        self.children.append(child)


class ChildNode:
    '''
    stand for attribute
    '''
    def __init__(self, name, type, xmltag="-", jsontag="-"):
        self.name = name
        self.type = type
        self.xmltag = xmltag
        self.jsontag = jsontag

    def __str__(self):
        return self.name


def file2cib_elem(f):
    '''
    open a xml file and get the root element
    '''
    cib_elem = None
    with open(f, 'r') as fd:
        try:
            cib_elem = etree.parse(fd).getroot()
        except Exception as err:
            print(err)
            return None

    # https://stackoverflow.com/questions/18159221/remove-namespace-and-prefix-from-xml-in-python-using-lxml
    if cib_elem is not None:
        for elem in cib_elem.getiterator():
            if not hasattr(elem.tag, 'find'):
                continue
            i = elem.tag.find('}')
            if i >= 0:
                elem.tag = elem.tag[i+1:]
        objectify.deannotate(cib_elem, cleanup_namespaces=True)
    return cib_elem


def handle_child(allNodes, node, rng=None, elem=None, root=None, child_type=None, xmltag="", jsontag=""):

    if rng:
        rng_file = os.path.join("/usr/share/pacemaker", rng)
        if not os.path.exists(rng_file):
            print("Error: %s not exists!" % rng_file)
            sys.exit(-1)

        root = file2cib_elem(rng_file)
        if root is None:
            print("Error: Parse %s failed!" % f)
            sys.exit(-1)

        elem = root


    for item in elem.iterchildren():
        name = item.get("name")
        if name and name.endswith("-unsupported"):
            # skip unsupported item
            continue
        href = item.get("href")

        # store old values
        old_childtype = child_type
        old_xmltag = xmltag
        old_jsontag = jsontag

        if item.tag in ("start", "interleave", "optional", "choice",
                        "zeroOrMore", "group", "grammar", "oneOrMore"):
            if item.tag in ("optional", "choice", "zeroOrMore"):
                # maybe not exists in cib, so json tag should be "omitempty"
                if "omitempty" not in jsontag:
                    jsontag = ",omitempty" + jsontag
            if item.tag in ("zeroOrMore", "oneOrMore"):
                # maybe more then one, so use "slice" to store them
                child_type = "slice"
            # recursively find thie item's child element
            handle_child(allNodes, node, elem=item, root=root, child_type=child_type, xmltag=xmltag, jsontag=jsontag)


        if item.tag == "element":
            if name is None:
                continue

            xmltag = name + xmltag
            jsontag = name + jsontag

            if child_type != "slice":
                child_type = name

            # add this element tag as this parent element node's children
            node.append(ChildNode(name, child_type, xmltag, jsontag))
            # create a new node(go class) for an element tag
            new_node = Node(name)
            if not node_exists(allNodes, new_node):
                # append this new node to the global node list
                allNodes.append(new_node)
                # recursively collect this new node's children list
                handle_child(allNodes, new_node, elem=item, root=root)


        if item.tag == "attribute":
            if name is None:
                continue

            child_type = "string"
            xmltag = ",attr" + xmltag
            xmltag = name + xmltag
            jsontag = name + jsontag

            # add this attribute tag as this element node's children
            node.append(ChildNode(name, child_type, xmltag, jsontag))


        if item.tag == "ref":
            if name is None or root is None:
                continue

            for elem in root.getiterator():
                ename = elem.get('name')
                if elem.tag == "define" and ename and ename == name:
                    # recursively find the element in 'ref' tag
                    handle_child(allNodes, node, elem=elem, root=root, child_type=child_type, xmltag=xmltag, jsontag=jsontag)


        if item.tag in ("include", "externalRef"):
            if href is None or not href.endswith(".rng"):
                continue
            # recursively read the related rng file
            handle_child(allNodes, node, rng=href, child_type=child_type, xmltag=xmltag, jsontag=jsontag)

        # recover old values
        childtype = old_childtype
        xmltag = old_xmltag
        jsontag = old_jsontag


def gen_struct(f):

    root = file2cib_elem(f)
    if root is None:
        return -1

    allNodes = []

    # start from pacemaker.rng file and the cib element
    for elem in root.getiterator():
        name = elem.get('name')
        if name and name == "cib":
            node = Node("cib")
            allNodes.append(node)
            handle_child(allNodes, node, elem=elem)
            break


    res = """
    package main

    import (
        "encoding/xml"
)

    """
    for node in allNodes:
        env = Environment(trim_blocks=True)
        env.globals['convert_name'] = convert_name
        res += env.from_string(goStructTemplate).render(node=node) + "\n\n"
    res += otherStructTemplate

    with open("api_structs.go", 'w') as f:
        f.write(res)
    run_cmd("go fmt api_structs.go")
    return 0


def run_cmd(cmd):
    try:
        proc = subprocess.Popen(cmd,
				shell=True,
				stdout=subprocess.PIPE,
				stderr=subprocess.PIPE)
        proc.communicate()
    except Exception as err:
        print(err)
    finally:
        return proc.returncode


def run(): 
    cmd = "rpm -q pacemaker-cli"
    rc = run_cmd(cmd)
    if rc != 0:
        print("Error: Please install pacemaker-cli first!")
        sys.exit(rc)

    start_file = "/usr/share/pacemaker/pacemaker.rng"
    if not os.path.exists(start_file):
        print("Error: %s not exists!" % start_file)
        sys.exit(-1)
    
    rc = gen_struct(start_file)
    if rc != 0:
        print("Error: gen_struct for %s failed!" % start_file)
        sys.exit(rc)

if __name__ == '__main__':
   run()
