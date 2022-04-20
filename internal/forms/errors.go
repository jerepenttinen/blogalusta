package forms

import "fmt"

type errors map[string][]string

func (e errors) Add(field, message string) {
	e[field] = append(e[field], message)
}

func (e errors) Get(field string) string {
	es := e[field]
	if len(es) == 0 {
		return ""
	}
	return es[0]
}

func (e errors) Has(field string) bool {
	_, ok := e[field]
	return ok
}

func (e errors) All() string {
	return fmt.Sprintf("%v", e)
}
