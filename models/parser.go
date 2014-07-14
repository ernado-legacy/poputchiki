package models

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	ContentTypeHeader = "Content-Type"
)

func mapToStruct(q url.Values, val interface{}) error {
	nQ := make(map[string]interface{})
	for key, value := range q {
		if len(value) == 1 {
			v := value[0]
			vInt, err := strconv.Atoi(v)
			if err != nil {
				nQ[key] = v
			} else {
				nQ[key] = vInt
			}
		} else {
			nQ[key] = value
		}
	}
	j, err := json.Marshal(nQ)
	if err != nil {
		return err
	}
	return json.Unmarshal(j, val)

}

func Parse(r *http.Request, v interface{}) error {
	contentType := r.Header.Get(ContentTypeHeader)
	if strings.Index(contentType, "json") != -1 {
		decoder := json.NewDecoder(r.Body)
		return decoder.Decode(v)
	}
	if strings.Index(contentType, "form") != -1 {
		return mapToStruct(r.Form, v)
	}

	return mapToStruct(r.URL.Query(), v)
}
