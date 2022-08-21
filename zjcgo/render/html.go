package render

import (
	"github.com/zhengjingcheng/zjcgo/internal/bytesconv"
	"html/template"
	"net/http"
)

type HTML struct {
	Data       any
	Name       string
	Template   *template.Template
	IsTemplate bool
}

type HTMLRender struct {
	Template *template.Template
}

func (h *HTML) Render(w http.ResponseWriter) error {
	h.WriteContentType(w)
	//如果使用模板类型
	if h.IsTemplate {
		err := h.Template.ExecuteTemplate(w, h.Name, h.Data)
		return err
	}
	//如果不使用模板类型
	_, err := w.Write(bytesconv.StringToBytes(h.Data.(string)))
	return err
}
func (x *HTML) WriteContentType(w http.ResponseWriter) {
	writeContentType(w, "text/html;charset = utf-8")
}
