$version: "2"

namespace examples

service HelloService {
    version: "1.0"
    operations: [Hello]
}

///
/// A minimal hello world action
///
@http(method: "GET", uri: "/hello", code: 200)
@readonly
operation Hello {
    input := {
		@httpQuery("caller")
		caller: String = "Mystery Caller"
	}
    output := {
		@httpPayload
		greeting: String
	}
}
