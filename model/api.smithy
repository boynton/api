$version: "2"

namespace model

/// BaseType - All other types are derived from these.
enum BaseType {
    Null,
    Bool,
    Int8,
    Int16,
    Int32,
    Int64,
    Float32,
    Float64,
    Decimal,
    Blob,
    String,
    Timestamp,
    Array,
    Object,
    List,
    Map,
    Struct,
    Enum,
    Union
}

/// Identifier - a simple symbolic name that most programming languages can use, i.e. "Blah"
@pattern("^[a-zA-Z_][a-zA-Z_0-9]*$")
string Identifier

/// Namespace - A sequence of one or more names delimited by a '.', i.e. "foo.bar"
@pattern("(^[a-zA-Z_][a-zA-Z_0-9]*\\.)*[a-zA-Z_][a-zA-Z_0-9]*$")
string Namespace

/// AbsoluteIdentifier - an Identifier in a Namespace, i.e. "foo.bar#Blah".
@pattern("^([a-zA-Z_][a-zA-Z_0-9]*\\.)*[a-zA-Z_][a-zA-Z_0-9]*#[a-zA-Z_][a-zA-Z_0-9]*$")
string AbsoluteIdentifier

/// TypeDef - a structure defining a new type in this system. New types cannot be derived from
/// these, but this new type can be used to specify the type of members in aggregate types. TypeDef
/// could more properly be defined as a Union of various types, but this structure is more
/// convenient.
structure TypeDef {
    @required
    id: AbsoluteIdentifier

    @required
    base: BaseType

    comment: String

    minValue: BigDecimal

    maxValue: BigDecimal

    minSize: Long

    maxSize: Long

    pattern: String

    items: AbsoluteIdentifier

    keys: AbsoluteIdentifier

    fields: List

    elements: List

    tags: List
}

/// FieldDef - describes each field in a structure or union.
structure FieldDef {
    name: String

    type: AbsoluteIdentifier

    required: Boolean

    comment: String
}

/// EnumElement - describes each element of an Enum type
structure EnumElement {
    @required
    symbol: Identifier

    value: String

    type: AbsoluteIdentifier

    comment: String
}

/// OperationDef - describes an operation, including its HTTP bindings
structure OperationDef {
    @required
    id: AbsoluteIdentifier

    comment: String

    httpMethod: String

    httpUri: String

    input: OperationInput

    output: OperationOutput

    exceptions: List
}

/// OperationInput - the description of an operation input. It is similar to a Struct definition,
/// but with HTTP bindings.
structure OperationInput {
    id: AbsoluteIdentifier

    fields: List

    comment: String
}

/// OperationInputField - the description of an operation input field
structure OperationInputField {
    @required
    name: Identifier

    @required
    type: AbsoluteIdentifier

    required: Boolean

    comment: String

    httpHeader: Identifier

    httpQuery: Identifier

    httpPath: Boolean

    httpPayload: Boolean
}

/// OperationOutput - the description of an operation output. Similar to a Struct definition, but
/// with HTTP bindings. Also used for OperationExceptions.
structure OperationOutput {
    id: AbsoluteIdentifier

    httpStatus: Integer

    fields: List

    comment: String
}

/// OperationOutputField
structure OperationOutputField {
    @required
    name: Identifier

    @required
    type: AbsoluteIdentifier

    comment: String

    httpHeader: Identifier

    httpPayload: Boolean
}

/// ServiceDef - the definition of a service, consisting of Types and Operations
structure ServiceDef {
    @required
    id: AbsoluteIdentifier

    version: String

    comment: String

    types: List

    operations: List
}
