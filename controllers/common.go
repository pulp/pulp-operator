package controllers

import (
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
)

func parseSettings(settings interface{}, log logr.Logger) {

	fieldKind := reflect.TypeOf(settings).Kind()

	if fieldKind == reflect.Int {
		// just making note on how to "convert" (actually, it is not a convertion, but a type assertion) from interface{} to int
		log.Info(fmt.Sprint(settings.(int)))
	} else if fieldKind == reflect.String {
		log.Info(fmt.Sprint(settings))
	} else if fieldKind == reflect.Map {
		parseSettings(settings.(map[string]interface{}), log)
	}

}
