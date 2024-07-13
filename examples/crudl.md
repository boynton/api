
# CrudlService


A simple service to demonstrate CRUDL operations


- **service**: "CrudlService"
- **namespace**: "example"

### Operation Index

- [CreateItem(item) → (item)](#createitem)
- [GetItem(itemId) → (item)](#getitem)
- [UpdateItem(itemId, item) → (item)](#updateitem)
- [DeleteItem(itemId) → ()](#deleteitem)
- [ListItems(limit, skip) → (listing)](#listitems)

### Type Index

- [Item](#item) → _Struct_
- [AttributeList](#attributelist) → _List_
- [Attribute](#attribute) → _Struct_
- [ExceptionInfo](#exceptioninfo) → _Struct_
- [ItemListing](#itemlisting) → _Struct_
- [ItemList](#itemlist) → _List_

## Operations

### CreateItem

<pre>
/// Create an item
@http(method: "POST", uri: "/items", code: 201)
operation CreateItem {

	input := {
		@httpPayload
		@required
		item: [**Item**](#Item)
	}

	output := {
		@httpPayload
		item: <i><b><a href="#item">Item</a></b></i>
	}

	errors: [
		BadRequest
	]
}
</pre>

### GetItem

```
/// Get an existing item
@readonly
@http(method: "GET", uri: "/items/{itemId}", code: 200)
operation GetItem {
    input := {
        @required
        @httpLabel
        itemId: String
    }

    output := {
        @httpPayload
        item: Item
    }

    errors: [
        NotFound
    ]
}
```

### UpdateItem

```
/// Update an existing item
@idempotent
@http(method: "PUT", uri: "/items/{itemId}", code: 200)
operation UpdateItem {
    input := {
        @required
        @httpLabel
        itemId: String

        @required
        @httpPayload
        item: Item
    }

    output := {
        @httpPayload
        item: Item
    }

    errors: [
        NotFound
        BadRequest
    ]
}
```

### DeleteItem

```
/// Delete an existing item
@idempotent
@http(method: "DELETE", uri: "/items/{itemId}", code: 204)
operation DeleteItem {
    input := {
        @required
        @httpLabel
        itemId: String
    }

    errors: [
        NotFound
    ]
}
```

### ListItems

```
/// List existing items
@readonly
@http(method: "GET", uri: "/items", code: 200)
operation ListItems {
    input := {
        @httpQuery("limit")
        limit: Integer

        @httpQuery("skip")
        skip: String
    }

    output := {
        @httpPayload
        listing: ItemListing
    }
}
```


## Types


### Item

```

/// The Item resource itself
structure Item {
    /// the id of the item
    @required
    id: String

    /// when the item ast last modified
    modified: Timestamp

    /// attributes of the Item
    attributes: AttributeList
}
```


### AttributeList

```

list AttributeList {
    member: Attribute
}
```


### Attribute

```

structure Attribute {
    key: String

    val: String
}
```


### ExceptionInfo

```

/// Info for the body of an exception, what is encoded on the wire in exception responses
structure ExceptionInfo {
    message: String
}
```


### ItemListing

```

/// The ItemListing is a paginated segment of the collection of items, with an optional continuation token
structure ItemListing {
    @required
    items: ItemList

    next: String
}
```


### ItemList

```

/// A list of Items
list ItemList {
    member: Item
}
```


