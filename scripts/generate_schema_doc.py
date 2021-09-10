# This scripts generates schema documentation by querying a populated local database
#
# You must have APOC enabled and this configuration set in neo4j.conf:
#   dbms.security.procedures.unrestricted=apoc.*

import os
from jinja2 import Template
from neo4j import GraphDatabase

NEO4J_URI = os.environ.get("NEO4J_URI", "neo4j://localhost:7687")
NEO4J_USERNAME = os.environ.get("NEO4J_USERNAME", "neo4j")
NEO4J_PASSWORD = os.environ.get("NEO4J_PASSWORD", "")

#

TEMPLATE = """
# Schema

_This file is generated by `./scripts/generate_schema_doc.py`_

## Table of Contents
{% for label, data in schema | dictsort %}
- [{{label}}](#{{label|lower}})
{%- endfor %}

{% for label, data in schema | dictsort %}
## {{label}}

### Properties

| Key | Type |
| --- | --- |
{%- for key, type in data["properties"] | dictsort %}
| {{key}} | {{type}} |
{%- endfor %}

### Relationships

| Outbound | Inbound |
| --- | --- |
{%- for _ in data["relationships"]["outbound"] %}
| {{data["relationships"]["outbound"][loop.index-1]}} | {{data["relationships"]["inbound"][loop.index-1]}} |
{%- endfor %}
{% endfor %}
"""


def get_node_labels(session):
    """Returns all node labels in use in the database."""
    result = session.run("MATCH (n) RETURN DISTINCT labels(n) as label")
    labels = []
    for record in result:
        labels.append(record.get("label")[0])
    return labels


def get_node_properties(session, label):
    """Returns dictionary of properties for node label"""
    result = session.run(
        f"""
        MATCH (n:{label})
        WITH n
        RETURN DISTINCT( apoc.meta.cypher.types(n) )
        LIMIT 1
        """
    )
    return result.single()[0]


def get_outbound_rels(session, label):
    """Returns distinct types of relationships for given node"""
    result = session.run(
        f"MATCH (:{label})-[rel]->() RETURN DISTINCT TYPE(rel) AS type"
    )
    rels = []
    for record in result:
        rels.append(record.get("type"))
    return rels


def get_inbound_rels(session, label):
    """Returns distinct types of relationships for given node"""
    result = session.run(
        f"MATCH (:{label})<-[rel]-() RETURN DISTINCT TYPE(rel) AS type"
    )
    rels = []
    for record in result:
        rels.append(record.get("type"))
    return rels


def main():
    driver = GraphDatabase.driver(uri=NEO4J_URI, auth=(NEO4J_USERNAME, NEO4J_PASSWORD))
    session = driver.session()

    labels = get_node_labels(session)

    schema = {}
    for label in labels:
        schema[label] = {}

        # get relationships
        schema[label]["relationships"] = {}

        outbounds = get_outbound_rels(session, label)
        inbounds = get_inbound_rels(session, label)
        # we want these lists to have same length for display purposes
        max_length = max(len(outbounds), len(inbounds))
        outbounds.extend([" "] * (max_length - len(outbounds)))
        inbounds.extend([" "] * (max_length - len(inbounds)))

        schema[label]["relationships"]["outbound"] = outbounds
        schema[label]["relationships"]["inbound"] = inbounds

        # get properties
        props = get_node_properties(session, label)
        schema[label]["properties"] = props

    t = Template(TEMPLATE)

    with open("../docs/schema.md", "w") as f:
        print(t.render(schema=schema), file=f)


if __name__ == "__main__":
    main()

