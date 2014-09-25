package models

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

const (
	ContentTypeHeader = "Content-Type"
)

func mapToStruct(q url.Values, val interface{}) (bson.M, error) {
	nQ := make(map[string]interface{})
	t := reflect.ValueOf(val)
	st := t.Elem().Type()
	fields := st.NumField()
	for i := 0; i < fields; i++ {
		field := st.Field(i)
		key := field.Tag.Get("json")
		if key == "" {
			key = strings.ToLower(field.Name)
		}
		if strings.Index(key, ",") != -1 {
			key = strings.Split(key, ",")[0]
		}
		if q.Get(key) == "" {
			continue
		}
		value := q[key]
		if len(value) == 0 {
			continue
		}
		if field.Type.Name() == "bool" {
			v := strings.ToLower(value[0])
			nQ[key] = v == "true" || v == "1"
			continue
		}
		if field.Type.Kind() == reflect.Slice {
			if len(value) > 1 {
				nQ[key] = value
				continue
			}
			reader := csv.NewReader(bytes.NewBufferString(value[0]))
			record, err := reader.Read()
			if err != nil {
				return nQ, err
			}
			nQ[key] = record
			continue
		}
		if len(value) == 1 {
			v := value[0]
			if field.Type.Name() == "ObjectId" && bson.IsObjectIdHex(v) {
				nQ[key] = bson.ObjectIdHex(v)
			}
			vInt, err := strconv.Atoi(v)
			if err != nil || field.Type.Name() == "string" {
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
		return nQ, err
	}
	return nQ, json.Unmarshal(j, val)
}

func mapToStructValue(q url.Values, val interface{}) error {
	_, err := mapToStruct(q, val)
	return err
}

func Parse(r *http.Request, v interface{}) error {
	contentType := r.Header.Get(ContentTypeHeader)
	if strings.Index(contentType, "json") != -1 {
		decoder := json.NewDecoder(r.Body)
		return decoder.Decode(v)
	}
	if strings.Index(contentType, "form") != -1 {
		err := r.ParseForm()
		if err != nil {
			log.Println("[parser]", "parsing error", err)
		}
		if len(r.Form) > 0 {
			return mapToStructValue(r.Form, v)
		}
		if len(r.PostForm) > 0 {
			return mapToStructValue(r.PostForm, v)
		}
		if r.MultipartForm != nil && len(r.MultipartForm.Value) > 0 {
			return mapToStructValue(r.MultipartForm.Value, v)
		}
	}
	return mapToStructValue(r.URL.Query(), v)
}

func GetQuery(r *http.Request, v interface{}) (query bson.M, err error) {
	content := r.Header.Get(ContentTypeHeader)
	if strings.Index(content, "json") != -1 {
		decoder := json.NewDecoder(r.Body)
		if err = decoder.Decode(&query); err != nil {
			return
		}
		if err = convert(query, v); err != nil {
			return
		}
		return
	}
	if strings.Index(content, "form") == -1 {
		return query, errors.New("Blank input or incorrect ContentType header")
	}
	err = r.ParseForm()
	if err != nil {
		return
	}
	form := r.Form
	if len(r.PostForm) > 0 {
		form = r.PostForm
	}
	return mapToStruct(form, v)
}

type Parser struct {
	r *http.Request
}

func (p Parser) Parse(v interface{}) error {
	return Parse(p.r, v)
}

func (p Parser) Query(v interface{}) (query bson.M, err error) {
	return GetQuery(p.r, v)
}

func NewParser(r *http.Request) Parser {
	return Parser{r}
}
