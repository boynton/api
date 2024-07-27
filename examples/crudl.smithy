$version: "2"

namespace example

/// A simple service to demonstrate CRUDL operations
service CrudlService {
    resources: [
        ItemResource
    ]
}

/// The Item resource is the primary entity this example service manages.
/// It is identified by a single id that provided by the service at creation time.
resource ItemResource {
    identifiers: { itemId: String }
    create: CreateItem
    read: GetItem
    update: UpdateItem
    delete: DeleteItem
    list: ListItems
}

/// Create an item
@http(method: "POST", uri: "/items", code: 201)
operation CreateItem {
    input := {
        @required
        @httpPayload
        data: ItemData
    }

    output := {
        @httpPayload
        item: Item
    }

    errors: [
        BadRequest
    ]
}

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

/// An operation exception representing bad client input to a request
@error("client")
@httpError(400)
structure BadRequest {
    @httpPayload
    info: ExceptionInfo
}

/// An operation exception representing that a resource was not found
@error("client")
@httpError(404)
structure NotFound {
    @httpPayload
    info: ExceptionInfo
}

/// A list of Items
list ItemList {
    member: Item
}

/// The ItemListing is a paginated segment of the collection of items, with an optional continuation token
structure ItemListing {
    @required
    items: ItemList

    next: String
}

/// The Item resource itself
structure Item {
    /// the id of the item
    @required
    id: String

    @required
    title: String

    /// attributes of the Item
    attributes: AttributeList

    /// when the item was last last modified
    modified: Timestamp
}

/// Data to create an Item
structure ItemData {
    @required
    title: String
    attributes: AttributeList
}

/// Info for the body of an exception, what is encoded on the wire in exception responses
structure ExceptionInfo {
    message: String
}

list AttributeList {
    member: Attribute
}

structure Attribute {
    key: String
    val: String
}
