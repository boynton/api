$version: "2"

namespace model

/// BaseType - All other types are derived from these.
enum BaseType {
    Bool
    Int8
    Int16
    Int32
    Int64
    Float32
    Float64
    Integer
    Decimal
    Blob
    String
    Timestamp
    List
    Map
    Struct
    Enum
    Union
    Any
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

@mixin
structure GenericTraits {
    comment: String
    tags: StringList
}

list StringList {
    member: String
}

list AbsoluteIdentifierList {
    member: AbsoluteIdentifier
}

@mixin
structure TypeTraits with [GenericTraits] {
    minValue: BigDecimal
    maxValue: BigDecimal
    minSize: Long
    maxSize: Long
    required: Boolean
    pattern: String
    items: AbsoluteIdentifier
    keys: AbsoluteIdentifier
    fields: FieldDefList
    elements: EnumElementList
}

list FieldDefList {
    member: FieldDef
}

list EnumElementList {
    member: EnumElement
}

/// TypeDef - a structure defining a new type in this system. New types cannot be derived from
/// these, but this new type can be used to specify the type of members in aggregate types. TypeDef
/// could more properly be defined as a Union of various types, but this structure is more
/// convenient.
structure TypeDef with [TypeTraits] {
    @required
    id: AbsoluteIdentifier

    @required
    base: BaseType
}

/// Field - describes each field in a structure or union.
structure FieldDef with [TypeTraits] {
    @required
    name: Identifier

    @required
    type: AbsoluteIdentifier
}

/// Element - describes each element of an Enum type
structure EnumElement with [GenericTraits] {
    @required
    symbol: Identifier

    value: String

    // type: AbsoluteIdentifier //defaults to String. This is to accomodate IntEnums in Smithy?
}

/// ResourceDef - describes a resource, and its operations and sub-resources
structure ResourceDef with [GenericTraits] {
    @required
    id: AbsoluteIdentifier

    create: AbsoluteIdentifier

    read: AbsoluteIdentifier

    update: AbsoluteIdentifier

    delete: AbsoluteIdentifier

    list: AbsoluteIdentifier

    put: AbsoluteIdentifier //in smithy, but do we use it?

    collectionOperations: AbsoluteIdentifierList
	
    operations: AbsoluteIdentifierList

    resources: AbsoluteIdentifierList
}


/// OperationDef - describes an operation, including its HTTP bindings
structure OperationDef with [GenericTraits] {
    @required
    id: AbsoluteIdentifier

    httpMethod: String

    httpUri: String

    input: OperationInput

    output: OperationOutput

    exceptions: AbsoluteIdentifierList

	examples: OperationExampleList
}

list OperationOutputList {
    member: OperationOutput
}

list OperationExampleList {
    member: OperationExample
}

structure OperationExample {
    title: String

    input: Document

    output: Document

    error: OperationErrorExample
}

structure OperationErrorExample {
	shapeId: AbsoluteIdentifier

	output: Document
}

/// OperationInput - the description of an operation input. It is similar to a Struct definition,
/// but with HTTP bindings.
structure OperationInput with [GenericTraits] {
    id: AbsoluteIdentifier
    fields: OperationInputFieldList
}

list OperationInputFieldList {
    member: OperationInputField
}

/// OperationInputField - the description of an operation input field
structure OperationInputField with [TypeTraits] {
    @required
    name: Identifier

    @required
    type: AbsoluteIdentifier

    default: Document

    httpHeader: String

    httpQuery: Identifier

    httpPath: Boolean

    httpPayload: Boolean
}

/// OperationOutput - the description of an operation output. Similar to a Struct definition, but
/// with HTTP bindings. Also used for OperationExceptions.
structure OperationOutput with [GenericTraits] {
    id: AbsoluteIdentifier
    httpStatus: Integer
    fields: OperationOutputFieldList
}

list OperationOutputFieldList {
    member: OperationOutputField
}

/// OperationOutputField
structure OperationOutputField with [TypeTraits] {
    @required
    name: Identifier

    @required
    type: AbsoluteIdentifier

    httpHeader: String

    httpPayload: Boolean
}

list TypeDefList {
    member: TypeDef
}

list OperationDefList {
    member: OperationDef
}

list ResourceDefList {
    member: ResourceDef
}

/// ServiceDef - the definition of a service, consisting of Types and Operations
structure ServiceDef with [GenericTraits] {
    @required
    id: AbsoluteIdentifier

    version: String

    base: String

    types: TypeDefList

	resources: ResourceDefList
	
    operations: OperationDefList

    exceptions: OperationOutputList
}
