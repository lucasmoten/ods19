# events

Package defining event schemas we publish, including a GEM implementation,
a custom payload, and the ICS 500-27 audit schema. 

```json
{
    "payload": {
        "audit_event" : { }
    }
}

```

Events are published for all endpoints. The publisher implementation is defined under **services**,
and called from the **server** package in http handlers, such as `createObject`, etc.
