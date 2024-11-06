package internal

import (
	"bytes"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type ParseableObject[T any] struct {
	Object T
}

type ParseDataInput struct {
	Doc     *goquery.Selection
	Options map[string]string
}

type ParseDataParams struct {
	TableId      string
	Row          int
	Cell         int
	FullSelector string
	RowSelector  string
	CellSelector string
	DataSelector string
	Attr         string
}

func Parse[T any](obj T, url string) {
	log.Printf("getting %s\n", url)
	bs, err := GetPage(url)
	if err != nil {
		panic(err)
	}
	bs = []byte(strings.Replace(strings.Replace(string(bs), "<!--", "", -1), "-->", "", -1))

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bs))
	if err != nil {
		panic(err)
	}

	po := ParseableObject[T]{Object: obj}
	po.ParseData(ParseDataInput{Doc: doc.Selection})
}

func (b *ParseableObject[T]) ParseData(input ParseDataInput) {
	vo := reflect.ValueOf(b.Object)
	o := vo.Elem()
	isStructField := false

	if o.Kind() == reflect.Interface {
		isStructField = true
		o = reflect.New(vo.Elem().Elem().Type()).Elem()
	}

	for i := 0; i < o.NumField(); i++ {
		field := o.Type().Field(i)
		tags := field.Tag
		params := getParsingParams(tags, input.Options)

		if hasParams(params) {
			tableSelector := fmt.Sprintf("table#%s", params.TableId)
			val := getFieldValue(&input, params)

			switch o.Field(i).Kind() {
			case reflect.Float64:
				v, _ := strconv.ParseFloat(val, 64)
				o.Field(i).SetFloat(v)
			case reflect.Uint:
				v, _ := strconv.ParseUint(val, 10, 64)
				o.Field(i).SetUint(v)
			case reflect.Int:
				v, _ := strconv.ParseInt(val, 10, 64)
				o.Field(i).SetInt(v)
			case reflect.String:
				o.Field(i).SetString(val)
			case reflect.Struct:
				ifc := o.Field(i).Interface()

				newPo := ParseableObject[interface{}]{
					Object: &ifc,
				}
				newPo.ParseData(ParseDataInput{Doc: input.Doc, Options: map[string]string{
					"tableId":      tags.Get("tableId"),
					"row":          tags.Get("row"),
					"cell":         tags.Get("cell"),
					"fullSelector": tags.Get("fullSelector"),
					"rowSelector":  tags.Get("rowSelector"),
					"cellSelector": tags.Get("cellSelector"),
				}})

				o.Field(i).Set(reflect.ValueOf(newPo.Object))
			case reflect.Slice | reflect.Array:
				slice := reflect.MakeSlice(o.Field(i).Type(), 0, 0)

				input.Doc.Find(tableSelector).Find("tbody").Find(params.RowSelector).Each(func(j int, s *goquery.Selection) {
					ifc := reflect.New(o.Field(i).Type().Elem()).Elem().Interface()
					newPo := ParseableObject[interface{}]{
						Object: &ifc,
					}

					newPo.ParseData(ParseDataInput{Doc: input.Doc, Options: map[string]string{
						"tableId":      tags.Get("tableId"),
						"row":          strconv.Itoa(j),
						"cell":         tags.Get("cell"),
						"fullSelector": tags.Get("fullSelector"),
						"rowSelector":  tags.Get("rowSelector"),
						"cellSelector": tags.Get("cellSelector"),
					}})

					slice = reflect.Append(slice, reflect.ValueOf(newPo.Object))
				})

				o.Field(i).Set(slice)
			}
		}
	}

	if isStructField {
		newPo := o.Interface()
		b.Object = newPo.(T)
	}

	b.PostProcess()
}

func (b *ParseableObject[T]) PostProcess() {
	vo := reflect.ValueOf(b.Object)

	method := vo.MethodByName("PostProcess")
	if method != reflect.ValueOf(nil) {
		vo.MethodByName("PostProcess").Call(nil)
	}
}

func getFieldValue(input *ParseDataInput, params *ParseDataParams) (val string) {
	var valCell *goquery.Selection
	if params.FullSelector != "" {
		valCell = input.Doc.Find(params.FullSelector).Eq(0)
	} else {
		tableSelector := fmt.Sprintf("table#%s", params.TableId)
		valCell = input.Doc.Find(tableSelector).Find("tbody").Find(params.RowSelector).Eq(params.Row).Find(params.CellSelector).Eq(params.Cell)
	}

	if params.DataSelector != "" {
		valCell = valCell.Find(params.DataSelector).Eq(0)
	}

	if params.Attr != "" {
		val, _ = valCell.Attr(params.Attr)
	} else {
		val = valCell.Text()
	}

	return
}

func getParsingParams(tags reflect.StructTag, options map[string]string) (params *ParseDataParams) {
	tableId := tags.Get("tableId")
	row, _ := strconv.Atoi(tags.Get("row"))
	cell, _ := strconv.Atoi(tags.Get("cell"))
	fullSelector := tags.Get("fullSelector")
	rowSelector := tags.Get("rowSelector")
	cellSelector := tags.Get("cellSelector")
	dataSelector := tags.Get("dataSelector")
	attr := tags.Get("attr")

	if options["tableId"] != "" {
		tableId = options["tableId"]
	}

	if options["row"] != "" {
		row, _ = strconv.Atoi(options["row"])
	}

	if options["cell"] != "" {
		cell, _ = strconv.Atoi(options["cell"])
	}

	if options["fullSelector"] != "" {
		fullSelector = options["fullSelector"]
	}

	if options["rowSelector"] != "" {
		rowSelector = options["rowSelector"]
	}

	if options["cellSelector"] != "" {
		cellSelector = options["cellSelector"]
	}

	if options["dataSelector"] != "" {
		dataSelector = options["dataSelector"]
	}

	if options["attr"] != "" {
		attr = options["attr"]
	}

	/* Set defaults */
	if cellSelector == "" {
		cellSelector = "td"
	}
	if rowSelector == "" {
		rowSelector = "tr"
	}

	return &ParseDataParams{
		TableId:      tableId,
		Row:          row,
		Cell:         cell,
		FullSelector: fullSelector,
		RowSelector:  rowSelector,
		CellSelector: cellSelector,
		DataSelector: dataSelector,
		Attr:         attr,
	}
}

func hasParams(params *ParseDataParams) bool {
	if params.TableId != "" || params.FullSelector != "" || params.DataSelector != "" || params.Attr != "" {
		return true
	} else {
		return false
	}
}
