package printer

import (
	"fmt"
	"reflect"
	"strings"
)

func fillSpaces(s string, maxLen int) string {
	if len(s) == maxLen {
		return s
	}

	var b strings.Builder

	b.WriteString(s)

	for i := 0; i < maxLen-len(s); i++ {
		fmt.Fprintf(&b, "%s", " ")
	}

	return b.String()
}

func Table[T any](records []T) {
	if len(records) == 0 {
		return
	}

	maxSizes := make(map[string]int)
  var titles []string

	for count, r := range records {
		st := reflect.TypeOf(r)

		if st.Kind() != reflect.Struct {
			continue
		}

		for i := 0; i < st.NumField(); i++ {
			field := st.Field(i)
			name := field.Name
			value := reflect.ValueOf(&r).Elem().FieldByName(name).String()

      if count == 0 {
        titles = append(titles, name)
      }

			if len(value) > maxSizes[name] {
				maxSizes[name] = len(value)
			}
		}
	}

	var header strings.Builder


  var i int
	for _, key := range titles {
		fmt.Fprintf(&header, "%s ", fillSpaces(strings.ToUpper(key), maxSizes[key]))

    titles[i] = key
    i += 1
	}

	fmt.Println(header.String())

	for _, r := range records {
    var record strings.Builder

		for _, key := range titles {
			value := reflect.ValueOf(&r).Elem().FieldByName(key).String()
			fmt.Fprintf(&record, "%s ", fillSpaces(value, maxSizes[key]))
		}

    // TODO: print to stout
    println(record.String())
	}
}
