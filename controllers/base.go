package controllers

import (
	"encoding/json"
	"github.com/astaxie/beego"
	"github.com/ernado/poputchiki/models"
)

type BaseController struct {
	beego.Controller
	db       models.DataBase
	realtime models.RealtimeInterface
}

func (controller *BaseController) R(v interface{}) {
	controller.Ctx.Output.ContentType("json")
	coder := json.NewEncoder(controller.Ctx.ResponseWriter)
	coder.Encode(v)
}
