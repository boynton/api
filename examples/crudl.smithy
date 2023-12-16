$version: "2"

namespace example

/// A simple service to demonstrate CRUDL operations
service TestService {
    resources: [ItemResource]
}

resource ItemResource {
    identifiers: {
	    itemId: String
    },
	create: CreateItem,
	read: GetItem,
	update: UpdateItem,
	delete: DeleteItem,
	list: ListItems
}

/// Create an item
@http(method: "POST", uri: "/items", code: 201)
operation CreateItem {
   input := {
	 @required
	 @httpPayload
	 item: Item
   }
   output := {
     @httpPayload
     item: Item
   }
    errors: [BadRequest]
}

///Get an existing item
@readonly
@http(method: "GET", uri: "/items/{itemId}", code: 200)
operation GetItem {
   input := for ItemResource {
	 @required
	 @httpLabel
     $itemId
   }
   output := {
     @httpPayload
     item: Item
   }
    errors: [NotFound]
}

///Update an existing item
@http(method: "PUT", uri: "/items/{itemId}", code: 200)
operation UpdateItem {
    input := for ItemResource {
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
    errors: [NotFound, BadRequest]
}

///Delete an existing item
@idempotent
@http(method: "DELETE", uri: "/items/{itemId}", code: 204)
operation DeleteItem {
    input := for ItemResource {
        @required
        @httpLabel
		itemId: String
    }
    errors: [NotFound]
}

/// List existing items
@readonly
@paginated(
  inputToken: "skip",
  outputToken: "listing.next",
  items: "listing.items"
)
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

/// A list of Items
list ItemList {
    member: Item
}

/// The ItemListing is a paginated segment of the collection of items, with an optional continuation token
structure ItemListing {
    @required
    items: ItemList,

    next: String
}

/// The Item resource itself
structure Item {
    /// the id of the item
    @required
    id: String

    ///when the item ast last modified
    modified: Timestamp

    ///arbitrary data
    data: String
}

/// A specific Error representing bad client input to a request
@httpError(400)
@error("client")
structure BadRequest {
    @httpPayload
    error: Error
}

/// A specific Error representing bad client input to a request
@httpError(404)
@error("client")
structure NotFound {
    @httpPayload
    error: Error
}

/// A generic Error entity, what is encoded on the wire in error responses
structure Error {
    message: String
}
