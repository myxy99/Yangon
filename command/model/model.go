package model

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"yangon/command/model/config"
	"yangon/pkg/database"
	"yangon/tools"
)

type List struct {
	Field   string `gorm:"Field"`
	Type    string `gorm:"Type"`
	Null    string `gorm:"Null"`
	Key     string `gorm:"Key"`
	Default string `gorm:"Default"`
	Extra   string `gorm:"Extra"`
}

//todo 生成handle map
//todo server map
//todo map

func (options *RunOptions) Run() {
	var err error
	//解析配置
	cfg, err := config.TryLoadFromDisk()
	tools.MustCheck(err)
	//链接数据库
	db, err := database.NewDatabaseClient(cfg.Mysql, nil)
	tools.MustCheck(err)
	//拉取模板
	tools.MustCheck(tools.GitClone("https://github.com/myxy99/Yangon-tpl.git", "tmp\\"+options.ProjectName))
	//defer删除拉取的模板
	defer tools.RemoveAllList("tmp")
	//查找表
	rows, err := db.DB().Raw("show tables;").Rows()
	tools.MustCheck(err)
	defer rows.Close()
	var table string
	for rows.Next() {
		tools.MustCheck(rows.Scan(&table))
		//把表名进行驼峰式转换
		modelName := tools.StrFirstToUpper(table)
		//查字段名
		listRows, err := db.DB().Raw(fmt.Sprintf("show columns from %s;", table)).Rows()
		tools.MustCheck(err)
		var TableFieldList, TableFieldMap string
		isTime := false
		Id := "ID"
		for listRows.Next() {
			list := new(List)
			//查询所有字段
			_ = listRows.Scan(&list.Key, &list.Type, &list.Default, &list.Extra, &list.Field, &list.Null)
			//把字段名进行驼峰式转换
			upper := tools.StrFirstToUpper(list.Key)
			//判断是不是主键
			if tools.IsPRI(list.Type) {
				Id = upper
			}
			var structType string
			var tmpIsTime bool
			//把对应的字段类型转换为结构体类型
			structType, tmpIsTime = tools.SqlType2StructType(list.Type)
			isTime = tmpIsTime || isTime
			//组合结构体中的字段，字符串
			TableFieldList += fmt.Sprintf("%s\t%s\n\t", upper, structType)
			if !tmpIsTime {
				TableFieldMap += fmt.Sprintf("%s\t%s\t `form:\"%s\" json:\"%s\" validate:\"required\"` \n\t", upper, structType, list.Key, list.Key)
			}
		}
		// model
		{
			var modelText string
			//获取到模板文件
			modelTpl, err := ioutil.ReadFile(fmt.Sprintf(`tmp/%s/model/model.go`, options.ProjectName))
			tools.MustCheck(err)
			//模板替换
			modelText = tools.ReplaceAllData(string(modelTpl), map[string]string{
				"{{TableFieldList}}": TableFieldList,
				"{{ProjectName}}":    options.ProjectName,
				"{{appName}}":        options.AppName,
				"{{TableName}}":      modelName,
				"{{tableName}}":      table,
				"{{ID}}":             Id,
			})
			//是否使用了time 包
			if isTime {
				modelText = strings.ReplaceAll(modelText, "{{IsTime}}", "\"time\"")
			} else {
				modelText = strings.ReplaceAll(modelText, "{{IsTime}}", "")
			}
			//模板替换文件夹位置
			modelPath := `internal/{{appName}}/model/{{table}}`
			modelPath = tools.ReplaceAllData(modelPath, map[string]string{
				"{{appName}}": options.AppName,
				"{{table}}":   table,
			})
			//创建文件夹
			tools.MustCheck(os.MkdirAll(modelPath, 777))
			//模板替换文件位置
			modelFile := `{{path}}/{{table}}.go`
			modelFile = tools.ReplaceAllData(modelFile, map[string]string{
				"{{path}}":  modelPath,
				"{{table}}": table,
			})
			//判断文件存在，如果存在 就备份之前文件
			if tools.CheckFileIsExist(modelFile) {
				tools.MustCheck(os.Rename(modelFile, fmt.Sprintf("%s.bak", modelFile)))
			}
			//向文件中写入数据
			tools.WriteToFile(modelFile, modelText)
			fmt.Println("model\t=>\t", modelFile)
		}

		//handle
		{
			var handleText string
			//获取到模板文件
			handleTpl, err := ioutil.ReadFile(fmt.Sprintf(`tmp/%s/model/handle.go`, options.ProjectName))
			tools.MustCheck(err)
			handleText = tools.ReplaceAllData(string(handleTpl), map[string]string{
				"{{ProjectName}}": options.ProjectName,
				"{{appName}}":     options.AppName,
				"{{AppName}}":     tools.StrFirstToUpper(options.AppName),
				"{{TableName}}":   modelName,
				"{{tableName}}":   table,
			})
			//模板替换文件位置
			handleFile := `internal/{{appName}}/api/v1/handle/{{table}}.go`
			handleFile = tools.ReplaceAllData(handleFile, map[string]string{
				"{{appName}}": options.AppName,
				"{{table}}":   table,
			})
			//判断文件存在，如果存在 就备份之前文件
			if tools.CheckFileIsExist(handleFile) {
				tools.MustCheck(os.Rename(handleFile, fmt.Sprintf("%s.bak", handleFile)))
			}
			//向文件中写入数据
			tools.WriteToFile(handleFile, handleText)
			fmt.Println("handle\t=>\t", handleFile)
		}

		//server
		{
			var serverText string
			//获取到模板文件
			serverTpl, err := ioutil.ReadFile(fmt.Sprintf(`tmp/%s/model/server.go`, options.ProjectName))
			tools.MustCheck(err)
			serverText = tools.ReplaceAllData(string(serverTpl), map[string]string{
				"{{ProjectName}}": options.ProjectName,
				"{{appName}}":     options.AppName,
				"{{AppName}}":     tools.StrFirstToUpper(options.AppName),
				"{{TableName}}":   modelName,
				"{{tableName}}":   table,
				"{{Id}}":          Id,
				"{{id}}":          tools.UnStrFirstToUpper(Id),
			})
			//模板替换文件位置
			//模板替换文件夹位置
			serverPath := `internal/{{appName}}/server/{{table}}`
			serverPath = tools.ReplaceAllData(serverPath, map[string]string{
				"{{appName}}": options.AppName,
				"{{table}}":   table,
			})
			//创建文件夹
			tools.MustCheck(os.MkdirAll(serverPath, 777))
			//模板替换文件位置
			serverFile := `{{path}}/{{table}}.go`
			serverFile = tools.ReplaceAllData(serverFile, map[string]string{
				"{{path}}":  serverPath,
				"{{table}}": table,
			})
			//判断文件存在，如果存在 就备份之前文件
			if tools.CheckFileIsExist(serverFile) {
				tools.MustCheck(os.Rename(serverFile, fmt.Sprintf("%s.bak", serverFile)))
			}
			//向文件中写入数据
			tools.WriteToFile(serverFile, serverText)
			fmt.Println("server\t=>\t", serverFile)
		}

		//registry
		{
			var registryText string
			//获取到模板文件
			registryTpl, err := ioutil.ReadFile(fmt.Sprintf(`tmp/%s/model/registry.go`, options.ProjectName))
			tools.MustCheck(err)
			registryText = tools.ReplaceAllData(string(registryTpl), map[string]string{
				"{{ProjectName}}": options.ProjectName,
				"{{appName}}":     options.AppName,
				"{{TableName}}":   modelName,
				"{{tableName}}":   table,
			})
			//模板替换文件位置
			registryFile := `internal/{{appName}}/api/v1/registry/{{table}}.go`
			registryFile = tools.ReplaceAllData(registryFile, map[string]string{
				"{{appName}}": options.AppName,
				"{{table}}":   table,
			})
			//判断文件存在，如果存在 就备份之前文件
			if tools.CheckFileIsExist(registryFile) {
				tools.MustCheck(os.Rename(registryFile, fmt.Sprintf("%s.bak", registryFile)))
			}
			//向文件中写入数据
			tools.WriteToFile(registryFile, registryText)
			fmt.Println("registry\t=>\t", registryFile)
		}

		//map
		{
			var mapText string
			//获取到模板文件
			mapTpl, err := ioutil.ReadFile(fmt.Sprintf(`tmp/%s/model/map.go`, options.ProjectName))
			tools.MustCheck(err)
			mapText = tools.ReplaceAllData(string(mapTpl), map[string]string{
				"{{TableName}}":     modelName,
				"{{TableFieldMap}}": TableFieldMap,
			})
			//模板替换文件位置
			mapFile := `internal/{{appName}}/map/{{table}}.go`
			mapFile = tools.ReplaceAllData(mapFile, map[string]string{
				"{{appName}}": options.AppName,
				"{{table}}":   table,
			})
			//判断文件存在，如果存在 就备份之前文件
			if tools.CheckFileIsExist(mapFile) {
				tools.MustCheck(os.Rename(mapFile, fmt.Sprintf("%s.bak", mapFile)))
			}
			//向文件中写入数据
			tools.WriteToFile(mapFile, mapText)
			fmt.Println("map\t=>\t", mapFile)
		}

		//关闭sql链接
		listRows.Close()
	}
}
