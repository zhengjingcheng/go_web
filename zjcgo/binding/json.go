package binding

import (
	"encoding/json"
	"errors"
	"net/http"
)

/*
·············································参数提取模块（第三方校验功能）·····················································
*/
//json绑定器
type jsonBinding struct {
}

var Validator StructValidator = &defaultValidator{}

func (jsonBinding) Name() string {
	return "json"
}
func (jsonBinding) Bind(r *http.Request, data any) error {
	body := r.Body
	if r == nil || body == nil {
		return errors.New("invalid request")
	}
	decoder := json.NewDecoder(body)
	err := decoder.Decode(data)
	if err != nil {
		return err
	}
	return validate(data)
}
