package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	v2 "github.com/bacalhau-project/bacalhau/pkg/config/types/v2"
)

// generateConstants is a recursive function that iterates through the fields of a given reflect.Type representing
// the structure of a configuration object. It constructs constant definitions for each field, building up the
// constant name and value based on the field's name, any associated "config" tags, and a provided prefix string.
//
// The parameters are:
// - t: The reflect.Type of the current part of the structure being examined.
// - prefix: A string that is used as the beginning of the constant names and values, reflecting the path within the structure.
// - file: An *os.File that represents the file to which the constant definitions are written.
//
// The function processes each field as follows:
// - If the field has a "config" tag, the prefix is replaced with the tag's value.
// - If the field is an anonymous field, the prefix remains unchanged.
// - If the field's name is "Node" and the prefix is already "Node", the prefix remains unchanged to avoid duplication.
// - In other cases, the field's name is appended to the prefix.
//
// If the field is itself a struct, the function calls itself recursively to handle the nested fields. If the field
// is not a struct, the function generates a line of code defining a constant, using the built-up prefix to form the
// constant's name and value. This line is written to the provided file.
//
// The resulting file will contain a series of constant definitions that can be used to access configuration values
// by name.
func generateConstants(t reflect.Type, prefix string, file *os.File) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("config")
		newPrefix := ""

		if tag != "" {
			// If there's a tag, we use it as the new prefix, discarding the old one
			newPrefix = tag
		} else if field.Anonymous {
			// For anonymous fields, keep the existing prefix
			newPrefix = prefix
		} else {
			// If prefix is empty, just use the field name. Otherwise, concatenate with "."
			newPrefix = prefix
			if newPrefix != "" {
				newPrefix += "."
			}
			newPrefix += field.Name
		}

		if field.Type.Kind() == reflect.Struct {
			// This is the modification where we add an intermediary path
			// constant before diving deeper into the struct.
			constantNameForStruct := strings.ReplaceAll(newPrefix, ".", "")
			fmt.Fprintf(file, "const %s = \"%s\"\n", constantNameForStruct, newPrefix)

			generateConstants(field.Type, newPrefix, file)
		} else {
			constantName := strings.ReplaceAll(newPrefix, ".", "")
			constantValue := newPrefix
			fmt.Fprintf(file, "const %s = \"%s\"\n", constantName, constantValue)
		}
	}
}

func main() {
	config := v2.Bacalhau{}

	// Open a file for writing
	file, err := os.Create("v2/generated_constants.go")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Write the package declaration
	fmt.Fprintf(file, "// CODE GENERATED BY pkg/config/types/gen_paths DO NOT EDIT\n\n")
	fmt.Fprintf(file, "package types\n\n")

	generateConstants(reflect.TypeOf(config), "", file)
}
