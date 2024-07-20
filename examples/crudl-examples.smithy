$version: "2"

namespace example

apply CreateItem @examples([
    {
        title: "Create Item 1"
        input: {
            data: {
                "title": "Test Item 1",
            }
        }
        output: {
            item: {
                "id": "item1",
                "title": "Test Item 1",
            }
        }
    }
    {
        title: "Create Item Failure"
        input: {
            data: {
            }
        }
        error: {
            shapeId: BadRequest
            content: {
                info: {
                    message: "Cannot create Item. Missing required parameter: 'title'"
                }
            }
        },
        allowConstraintErrors: true
    }
])

apply GetItem @examples([
    {
        title: "Get Item 1"
        input: {
            itemId: "item1"
        }
        output: {
            item: {
                "id": "item1",
                "title": "Test Item 1",
            }
        }
    }
])

apply ListItems @examples([
    {
        title: "Listing page 1"
        input: {
            "limit": 2
        }
        output: {
            listing: {
                "items": [
                    {
                        "id": "item1",
                            "title": "Test Item 1",
                    },
                    {
                        "id": "item2",
                            "title": "Test Item 2",
                    },
                ],
                "next": "32vg321"
            }
        }
    }
    {
        title: "Listing page 2"
        input: {
            "limit": 2,
            "skip": "32vg321"
        }
        output: {
            listing: {
                "items": [
                    {
                        "id": "item3",
                            "title": "Test Item 3",
                    },
                ],
            }
        }
    }
])
