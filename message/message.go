package message

import (
	"errors"
	"strings"
)

var (
	InvalidInput      = errors.New("Invalid input")
	InvalidTags       = errors.New("Cannot parse tags, invalid input")
	InvalidPrefix     = errors.New("Cannot parse prefix, invalid input")
	InvalidCommand    = errors.New("Cannot parse command, invalid input")
	InvalidParam      = errors.New("Cannot parse param, invalid input")
	InvalidCharacter  = errors.New("Invalid character detected")
	MissingCRLF       = errors.New("Missing CarriageReturn LineFeed")
	MiddleParamToLong = errors.New("Middle Param to long")
)

type Message struct {
	Tags []string
	Prefix string
	Command string
	Params []string
}

func ParseMessage(input string) (*Message, error) {
	chars := []byte(input)

	// FIXME: input already arrive without CRLF
	size := len(chars)
	// if size <= 2 {
	// 	return nil, InvalidInput
	// }
	// if chars[size - 2] != '\r' && chars[size - 1] != '\n' {
	// 	return nil, MissingCRLF
	// }
	// limit := size - 2
	limit := size

	return parseTags(chars, limit, &Message{
		Tags: []string{},
		Prefix: "",
		Command: "",
		Params: []string{},
	})
}

func parseTags(chars []byte, limit int, msg *Message) (*Message, error) {
	start := 0
	if chars[start] == '@' {
		var index int
		for index = 1; index < limit && chars[index] != ' '; index++ {

		}
		if index == limit || index <= 1 {
			return nil, InvalidTags
		}
		tags := strings.Split(string(chars[1:index]), ";")
		msg.Tags = tags
		start = index + 1
	}
	return parsePrefix(chars, start, limit, msg)
}

func parsePrefix(chars []byte, start, limit int, msg *Message) (*Message, error) {
	offset := -1
	for i := start; i < limit; i++ {
		if chars[i] != ' ' {
			start = i
			break
		}
	}
	if chars[start] == ':' {
		for i := start; i < limit; i++ {
			if chars[i] == ' ' { // ASCII SPACE
				offset = i
				break
			}
		}
		// If there is a prefix then it should find a SPACE
		// Note that a Message with only PREFIX is not a valid message
		if offset <= 1 {
			return nil, InvalidPrefix
		}

		msg.Prefix = string(chars[start + 1:offset]) // discard ':'
		start = offset + 1
		if start == limit {
			return nil, InvalidInput
		}
	}

	return parseCommand(chars, start, limit, msg)
}

func parseCommand(chars []byte, start, limit int, msg *Message) (*Message, error) {
	if start == limit {
		return nil, InvalidCommand
	}

	if isDigit(chars[start]) {
		// 3digit command, in gnu regex syntax [0-9]{3}
		if limit < start + 3 {
			return nil, InvalidCommand
		}
		for i := 0; i < 3; i++ {
			if !isDigit(chars[start + i]) {
				return nil, InvalidCommand
			}
		}
		msg.Command = string(chars[start:start+3])
		start = start + 4 // discard space after command
	} else {
		// 1*letter commmand, in gnu regex syntax letter{1,*}
		offset := -1
		var i int
		for i = start; i < limit; i++ {
			if chars[i] == ' ' {
				offset = i
				break
			}
			if !isLetter(chars[i]) {
				return nil, InvalidCommand
			}
		}

		if offset == -1 && offset == start {
			return nil, InvalidCommand
		}
		if i == limit {
			msg.Command = string(chars[start:limit])
			return msg, nil
		} else {
			msg.Command = string(chars[start:offset])
			// discard Space after command
			start = offset + 1
		}
	}

	return parseParams(chars, start, limit, msg)
}

func parseParams(chars []byte, start, limit int, msg *Message) (*Message, error) {
	params := []string{}

	remaining := chars[start:limit]
	size := len(remaining)
	for i := 0; i < size; i++ {
		if remaining[i] == ':' {
			// read trailing
			if i + 1 < size {
				params = append(params, string(remaining[i + 1:]))
			}
			break
		} else if isNoSpCrLfCl(remaining[i]) {
			// read middle
			var j int
			for j = i + 1; j < size && (isNoSpCrLfCl(remaining[j]) || remaining[j] == ':') ; j++ {

			}
			if j - 1 + 1 > 14 || len(params) >= 14 {
				return nil, InvalidParam
			}
			params = append(params, string(remaining[i:j]))
			i = j
		}
	}

	msg.Params = params

	return msg, nil
}

// only ascii digit
func isDigit(char byte) bool {
	return char >= '0' && char <= '9'
}

// only ascii character
func isLetter(char byte) bool {
	return char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z'
}

// esoteric name:
// from RFC 2812
// nospcrlfcl = %x01-09 | %x0B-0C | %x0E-1F | %x21-39 | %x3B-FF
// nospcrlfcl = an ascci character in the above ranges
// nospcrlfcl => No (Space | CarriageReturn+LineFeed | Column)
func isNoSpCrLfCl(char byte) bool {
	return char != 0 && char != 0x0A && char != 0x0D && char != 0x20 && char != 0x3A
}
