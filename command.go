package main

func handleCommand(value Value) Value {
	if value.typ != '*' {
		return Value{}
	}
	if len(value.array) == 0 {
		return Value{}
	}
	if value.array[0].typ != '$' {
		return Value{}
	}
	command := value.array[0].str

	if command == "PING" {
		return Value{
			typ: '+',
			str: "PONG",
		}

	}
	return Value{}

}
