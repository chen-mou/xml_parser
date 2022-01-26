package xml

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"xml_parser/lib/tool"
)

var tags = [2]string{"type", "suffix"}

type label struct {
	suffix string
	tag    string
	value  string
	name   string
	field  map[string]string
}

type field struct {
	t      reflect.Type
	v      reflect.Value
	suffix string
	fields map[string]*reflect.Value
	name   string
}

func ObjectToXmlStr(value interface{}) string {
	val := reflect.ValueOf(value)
	t := reflect.TypeOf(value)
	name := t.Name()
	l := label{
		name:  name,
		field: map[string]string{},
	}
	l.value = objectToXmlStr(val, t, &l)
	l.name = strings.ToLower(l.name[0:1]) + l.name[1:]
	str, _ := tool.String("<{{.Name}}{{range $i,$v := .Map}} {{$i}}={{$v}}{{end}}>{{.Value}}</{{.Name}}>").Format(map[string]interface{}{
		"Name":  l.name,
		"Value": l.value,
		"Map":   l.field,
	})
	return str
}

func objectToXmlStr(v reflect.Value, t reflect.Type, parent *label) string {
	str := ""
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		vField := v.Field(i)
		if !vField.IsValid() {
			continue
		}
		l := label{
			suffix: "",
			field:  map[string]string{},
		}
		l.name = field.Name
		l.value = getValue(field.Type, vField, &l)
		ok := handlerTag(field, vField, &l, parent)
		if ok {
			continue
		}
		l.name = strings.ToLower(l.name[0:1]) + l.name[1:]
		s, _ := tool.String(`<{{.Name}}{{.Suffix}}{{range $i,$v := .Map}} {{$i}}={{$v}}{{end}}>{{.Value}}</{{.Name}}{{.Suffix}}>`).Format(map[string]interface{}{
			"Name":   l.name,
			"Value":  l.value,
			"Suffix": l.suffix,
			"Map":    l.field,
		})
		str += s
	}
	return str
}

func handlerTag(t reflect.StructField, value reflect.Value, now *label, parent *label) bool {
	for _, v := range tags {
		val, ok := t.Tag.Lookup(v)
		if !ok {
			continue
		}
		switch v {
		case "suffix":
			now.suffix = val
			break
		case "type":
			switch val {
			case "LABEL_VALUE":
				t.Name = strings.ToLower(t.Name[0:1]) + t.Name[1:]
				parent.field[t.Name] = getValue(t.Type, value, nil)
				return true
			case "LABEL_CONTENT":
				structType, ok1 := t.Type.FieldByName("Value")
				if !ok1 {
					panic("")
				}
				now.value = getValue(structType.Type, value.FieldByName("Value"), now)
				if now.value == "" {
					return true
				}
				for i := 0; i < t.Type.NumField(); i++ {
					fieldName := t.Type.Field(i).Name
					if fieldName == "Value" {
						continue
					}
					handlerTag(t.Type.Field(i), value.Field(i), nil, now)
				}
			}

		}
	}
	return false
}

func getValue(t reflect.Type, value reflect.Value, l *label) string {
	typeName := t.String()
	if typeName == "string" || typeName == "bool" || typeName == "int" || typeName == "int64" || typeName == "int32" {
		switch typeName {
		case "string":
			return value.String()
		case "int", "int32", "int64":
			return strconv.FormatInt(value.Int(), 10)
		case "bool":
			return strconv.FormatBool(value.Bool())
		}
	} else {
		return objectToXmlStr(value, t, l)
	}
	return ""
}

func analyzeTag(field reflect.StructField, v reflect.Value, parent *field, now *field) bool {
	for _, tag := range tags {
		val, ok := field.Tag.Lookup(tag)
		if !ok {
			continue
		}
		switch tag {
		case "type":
			switch val {
			case "LABEL_VALUE":
				parent.fields[field.Name] = &v
				return true
			case "LABEL_CONTENT":
				now.v = v.FieldByName("Value")
				structField, ok1 := field.Type.FieldByName("Value")
				now.t = structField.Type
				if !ok1 {
					panic("please add value which name is \"Value\" in you struct")
				}
				for i := 0; i < field.Type.NumField(); i++ {
					temp := field.Type.Field(i)
					if temp.Name == "Value" {
						continue
					}
					mid := v.Field(i)
					analyzeTag(temp, mid, now, nil)
				}
			}
		case "suffix":
			now.suffix = val
			break
		}
	}
	return false
}

func XmlStrToObject(str string, o interface{}) {
	t := reflect.TypeOf(o).Elem()
	val := reflect.ValueOf(o).Elem()
	parent := field{
		fields: map[string]*reflect.Value{},
		v:      val,
		t:      t,
		suffix: "",
		name:   t.Name(),
	}
	getObject(str, &val, t, &parent)
	reg, _ := tool.String("<{{.Name}}(.*?)>(.*?)</{{.Name}}>").Format(map[string]interface{}{
		"Name": parent.name,
	})
	re := regexp.MustCompile(reg)
	strs := re.FindStringSubmatch(str)
	setField(parent.fields, strs[1])
}

func getObject(str string, val *reflect.Value, t reflect.Type, parent *field) reflect.Value {
	length := t.NumField()
	arr := make([]field, length)
	initArr(arr, t, val, parent)
	for i := 0; i < length; i++ {
		tempField := arr[i].v
		if arr[i].t == nil || !tempField.CanSet() {
			continue
		}
		reg, _ := tool.String("<{{.Name}}{{.Suffix}}(.*?)>(.*?)</{{.Name}}{{.Suffix}}>").Format(map[string]interface{}{
			"Name":   arr[i].name,
			"Suffix": arr[i].suffix,
		})
		re := regexp.MustCompile(reg)
		strs := re.FindStringSubmatch(str)
		value := strs[2]
		setValue(value, &tempField, &arr[i])
		setField(arr[i].fields, strs[1])
	}
	return *val
}

func initArr(arr []field, t reflect.Type, val *reflect.Value, parent *field) {
	for i, j := 0, 0; i < len(arr); i++ {
		fieldStruct := t.Field(i)
		v := val.Field(i)
		temp := field{
			t:      fieldStruct.Type,
			v:      v,
			fields: map[string]*reflect.Value{},
			suffix: "",
			name:   fieldStruct.Name,
		}
		ok := analyzeTag(fieldStruct, v, parent, &temp)
		if ok {
			continue
		}
		typeName := fieldStruct.Type.Name()
		if typeName == "string" || typeName == "int" ||
			typeName == "int32" || typeName == "int64" ||
			typeName == "bool" || i == j {
			arr[i] = temp
			continue
		}
		arr[i] = arr[j]
		arr[j] = temp
		j++
	}
}

func setValue(value string, tempField *reflect.Value, parent *field) {
	switch parent.t.String() {
	case "string":
		tempField.SetString(value)
		break
	case "int32", "int64", "int":
		num, _ := strconv.ParseInt(value, 10, 64)
		tempField.SetInt(num)
		break
	case "bool":
		bo, _ := strconv.ParseBool(value)
		tempField.SetBool(bo)
		break
	default:
		tempField.Set(getObject(value, tempField, parent.t, parent))
	}
}

func setField(m map[string]*reflect.Value, str string) {
	fields := strings.Split(str, " ")
	for _, val := range fields {
		if val == "" {
			continue
		}
		nameAndValue := strings.Split(val, "=")
		value := m[nameAndValue[0]]
		temp := field{
			t: value.Type(),
		}
		setValue(nameAndValue[1], m[nameAndValue[0]], &temp)
	}
}
