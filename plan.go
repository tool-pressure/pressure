package main

import (
	"gopkg.in/mgo.v2/bson"
	"fmt"
	"log"
	"strconv"
	"net/http"

	"github.com/martini-contrib/render"
	"encoding/json"
	"time"
)

type Item struct {
	Type   string // boom ,vegeta
	Boom   BoomJob
	Vegeta VegetaJob
}

type PlanTask struct {
	Id bson.ObjectId `json:"id"        bson:"_id,omitempty"`
	Name     string
	// Http API Method ["GET", "POST" ...]
	Method  string
	Items   []Item
	LastRunTs int64
}

type PlanEditForm struct {
	Plan     *PlanTask
	Methods []MethodSelector
	Booms   []BoomJob
	Vegetas []VegetaJob
}

func GetPlans(req *http.Request, r render.Render) {
	var name = req.FormValue("name")
	var page = req.FormValue("p")

	var condition = bson.M{}
	if name != "" {
		condition["name"] = name
	}

	if len(condition) == 0 {
		condition = nil
	}
	total, err := G_MongoDB.C("plan_tasks").Find(condition).Count()
	if err != nil {
		log.Panic(err)
	}
	var pager = NewPager(20, total)
	pager.CurrentPage, err = strconv.Atoi(page)
	pager.UrlPattern = fmt.Sprintf("/plan/?p=%%d&name=%s", name)
	var tasks []PlanTask
	err = G_MongoDB.C("plan_tasks").Find(condition).Skip(pager.Offset()).Sort("-lastrunts").Limit(pager.Limit()).All(&tasks)
	if err != nil {
		log.Panic(err)
	}
	var context = make(map[string]interface{})
	context["tasks"] = tasks
	context["pager"] = pager
	RenderTemplate(r, "plan_tasks", context)
}



func CreateGetPlan(req *http.Request, r render.Render) {
	var name = req.FormValue("name")
	var task = PlanTask{
		Id:                 bson.NewObjectId(),
		Name:               name,
		LastRunTs:          time.Now().Unix(),
	}
	err := G_MongoDB.C("plan_tasks").Insert(&task)
	if err != nil {
		log.Panic(err)
	}
	r.Redirect(fmt.Sprintf("/plan/edit?plan_id=%s", task.Id.Hex()))
}

func EditPlanPage(req *http.Request, r render.Render) {
	var planId = req.FormValue("plan_id")
	var task PlanTask
	err := G_MongoDB.C("plan_tasks").FindId(bson.ObjectIdHex(planId)).One(&task)
	if err != nil {
		log.Panic(err)
	}
	var context = make(map[string]interface{})
	var form = PlanEditForm{Plan: &task}
	form.Methods = GenMethodSelectors(task.Method)


	// 获取所有的 boom 任务
	form.Booms = FindAllBooms()

	// 获取所有的 vegeta 任务
	form.Vegetas = FindAllVegetas()

	context["form"] = form

	RenderTemplate(r, "plan_edit", context)
}




func EditPlan(req *http.Request, r render.Render) {
	req.ParseForm()
	var planId = req.FormValue("plan_id")
	var task PlanTask
	err := G_MongoDB.C("plan_tasks").FindId(bson.ObjectIdHex(planId)).One(&task)
	if err != nil {
		log.Panic(err)
	}
	task.Name = req.FormValue("name")
	task.Method = req.FormValue("method")
	var headerSeeds = []map[string]interface{}{}
	var paramSeeds = []map[string]interface{}{}
	for _, header := range req.Form["header"] {
		var seed map[string]interface{}
		json.Unmarshal([]byte(header), &seed)
		headerSeeds = append(headerSeeds, seed)
	}
	for _, param := range req.Form["param"] {
		var seed map[string]interface{}
		json.Unmarshal([]byte(param), &seed)
		paramSeeds = append(paramSeeds, seed)
	}

	var changed = bson.M{
		"name":      task.Name,
		"method":    task.Method,
	}
	var op = bson.M{"$set": changed}
	err = G_MongoDB.C("plan_tasks").UpdateId(task.Id, op)
	if err != nil {
		log.Panic(err)
	}
	r.Redirect("/plan/")
}