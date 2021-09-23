package graphql

import (
	"fmt"

	"github.com/vektah/gqlparser/v2/ast"
)

// ApplyFragments takes a list of selections and merges them into one, embedding any fragments it
// runs into along the way
func ApplyFragments(selectionSet ast.SelectionSet, fragmentDefs ast.FragmentDefinitionList) (ast.SelectionSet, error) {
	collectedFieldSet, err := newFieldSet(selectionSet, fragmentDefs)
	return collectedFieldSet.ToSelectionSet(), err
}

// fieldSet is a unique set of fieldEntries. Adding an existing field with the same name (or alias) will merge the selectionSets.
type fieldSet map[string]*fieldEntry

// fieldEntry is a set entry that generates a copy of the field with a new ast.SelectionSet
type fieldEntry struct {
	field        *ast.Field // Never modify the pointer Field value. Only copy-on-update when converting back to ast.SelectionSet.
	selectionSet fieldSet
}

// Make creates a new ast.Field with this entry's new ast.SelectionSet
func (e fieldEntry) Make() *ast.Field {
	shallowCopyField := *e.field
	shallowCopyField.SelectionSet = e.selectionSet.ToSelectionSet()
	return &shallowCopyField
}

// newFieldSet converts an ast.SelectionSet into a unique set of ast.Fields by resolving all fragements.
// The fieldSet can then convert back to a fully-resolved ast.SelectionSet.
func newFieldSet(selectionSet ast.SelectionSet, fragments ast.FragmentDefinitionList) (fieldSet, error) {
	set := make(fieldSet)
	for _, selection := range selectionSet {
		if err := set.Add(selection, fragments); err != nil {
			return nil, err
		}
	}
	return set, nil
}

func (s fieldSet) Add(selection ast.Selection, fragments ast.FragmentDefinitionList) error {
	switch selection := selection.(type) {
	case *ast.Field:
		key := selection.Name
		if selection.Alias != "" {
			key = selection.Alias
		}

		entry, ok := s[key]
		if !ok {
			entry = &fieldEntry{
				field:        selection,
				selectionSet: make(fieldSet),
			}
			s[key] = entry
		}
		for _, subselect := range selection.SelectionSet {
			if err := entry.selectionSet.Add(subselect, fragments); err != nil {
				return err
			}
		}

	case *ast.InlineFragment:
		// each field in the inline fragment needs to be added to the selection
		for _, fragmentSelection := range selection.SelectionSet {
			// add the selection from the field to our accumulator
			if err := s.Add(fragmentSelection, fragments); err != nil {
				return err
			}
		}

	// fragment selections need to be unwrapped and added to the final selection
	case *ast.FragmentSpread:
		// grab the definition for the fragment
		definition := fragments.ForName(selection.Name)
		if definition == nil {
			// this shouldn't happen since validation has already ran
			return fmt.Errorf("could not find fragment definition: %s", selection.Name)
		}

		// each field in the inline fragment needs to be added to the selection
		for _, fragmentSelection := range definition.SelectionSet {
			// add the selection from the field to our accumulator
			if err := s.Add(fragmentSelection, fragments); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s fieldSet) ToSelectionSet() ast.SelectionSet {
	selectionSet := make(ast.SelectionSet, 0, len(s))
	for _, entry := range s {
		selectionSet = append(selectionSet, entry.Make())
	}
	return selectionSet
}

func SelectedFields(source ast.SelectionSet) []*ast.Field {
	fields := []*ast.Field{}
	for _, selection := range source {
		if field, ok := selection.(*ast.Field); ok {
			fields = append(fields, field)
		}
	}
	return fields
}

// ExtractVariables takes a list of arguments and returns a list of every variable used
func ExtractVariables(args ast.ArgumentList) []string {
	// the list of variables
	variables := []string{}

	// each argument could contain variables
	for _, arg := range args {
		extractVariablesFromValues(&variables, arg.Value)
	}

	// return the list
	return variables
}

func extractVariablesFromValues(accumulator *[]string, value *ast.Value) {
	// we have to look out for a few different kinds of values
	switch value.Kind {
	// if the value is a reference to a variable
	case ast.Variable:
		// add the ference to the list
		*accumulator = append(*accumulator, value.Raw)
	// the value could be a list
	case ast.ListValue, ast.ObjectValue:
		// each entry in the list or object could contribute a variable
		for _, child := range value.Children {
			extractVariablesFromValues(accumulator, child.Value)
		}
	}
}
