package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-redis/redis/v7"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/satori/go.uuid"
	"time"
)

// models
type Base struct {
	IsVaild    bool       `json:"is_valid" gorm:"COLUMN:is_valid;default:1"`
	Timestamp  int        `json:"timestamp" gorm:"column:timestamp;index"`
	CreateTime *time.Time `json:"create_time" gorm:"column:create_time;index;default:now()"`
	UpdateTime *time.Time `json:"update_time" gorm:"column:update_time;index;default:now()"`
}

type BaseModelUUIDPk struct {
	ID string `gorm:"column:object_id;primary_key;type:varchar(32)" json:"object_id"`
	Base
}

type OnLineStatus struct {
	Flag             string `json:"flag"`
	LastActivityTime string `json:"last_activity_time"`
}

type Device struct {
	BaseModelUUIDPk
	EnterpriseID string       `json:"enterprise_id" gorm:"COLUMN:enterprise_id;index;type:varchar(32)"`
	UUID         string       `json:"uuid" gorm:"COLUMN:uuid;index;type:varchar(100)"`
	DeviceName   string       `json:"device_name" gorm:"COLUMN:device_name;type:varchar(100)"`
	VoiceLevel   uint8        `json:"voice_level" gorm:"COLUMN:voice_level;default:1;type:smallint"`
	WifiName     string       `json:"wifi_name" gorm:"COLUMN:wifi_name;type:varchar(100)"`
	LanIP        string       `json:"lan_ip" gorm:"COLUMN:lan_ip;type:varchar(20)"`
	Mac          string       `json:"mac" gorm:"COLUMN:mac;type:varchar(50)"`
	Status       OnLineStatus `json:"status" gorm:"-"`
}

// response
func Succeed(data interface{}) (int, map[string]interface{}) {
	return http.StatusOK, gin.H{"code": 200, "data": data, "msg": "ok"}
}

func Response(code int, data interface{}, msg string) (int, map[string]interface{}) {
	return http.StatusOK, gin.H{"code": code, "data": data, "msg": msg}
}

// Init
func InitDB() *gorm.DB {
	if sql, err := gorm.Open("mysql", "root:dbadmin@tcp(172.27.106.1:3306)/daka_equipment?charset=utf8mb4&parseTime=true"); err == nil {
		//sql.SingularTable(true)
		return sql
	} else {
		panic(err)
		return nil
	}
}

func InitRedis() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "172.27.106.3:6379",
		Password: "troila",
		DB:       0,
	})
	return client
}

// utils
func Struct2Map(obj interface{}) (data map[string]interface{}, err error) {
	data = make(map[string]interface{})
	objT := reflect.TypeOf(obj)
	objV := reflect.ValueOf(obj)
	for i := 0; i < objT.NumField(); i++ {
		data[objT.Field(i).Name] = objV.Field(i).Interface()
	}
	err = nil
	return
}

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method               //请求方法
		origin := c.Request.Header.Get("Origin") //请求头部
		var headerKeys []string                  // 声明请求头keys
		for k, _ := range c.Request.Header {
			headerKeys = append(headerKeys, k)
		}
		headerStr := strings.Join(headerKeys, ", ")
		if headerStr != "" {
			headerStr = fmt.Sprintf("access-control-allow-origin, access-control-allow-headers, %s", headerStr)
		} else {
			headerStr = "access-control-allow-origin, access-control-allow-headers"
		}
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Origin", "*")                                       // 这是允许访问所有域
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE,UPDATE") //服务器支持的所有跨域请求的方法,为了避免浏览次请求的多次'预检'请求
			//  header的类型
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Length, X-CSRF-Token, Token,session,X_Requested_With,Accept, Origin, Host, Connection, Accept-Encoding, Accept-Language,DNT, X-CustomHeader, Keep-Alive, User-Agent, X-Requested-With, If-Modified-Since, Cache-Control, Content-Type, Pragma")
			//              允许跨域设置                                                                                                      可以返回其他子段
			c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers,Cache-Control,Content-Language,Content-Type,Expires,Last-Modified,Pragma,FooBar") // 跨域关键设置 让浏览器可以解析
			c.Header("Access-Control-Max-Age", "172800")                                                                                                                                                           // 缓存请求信息 单位为秒
			c.Header("Access-Control-Allow-Credentials", "false")                                                                                                                                                  //  跨域请求是否需要带cookie信息 默认设置为true
			c.Set("content-type", "application/json")                                                                                                                                                              // 设置返回格式是json
		}

		//放行所有OPTIONS方法
		if method == "OPTIONS" {
			c.JSON(http.StatusOK, "Options Request!")
		}
		// 处理请求
		c.Next() //  处理请求
	}
}

// db func
func BeforeCreate(scope *gorm.Scope) {
	var u uuid.UUID
	u = uuid.NewV4()
	uStr := strings.Replace(u.String(), "-", "", -1)
	scope.SetColumn("object_id", uStr)
	scope.SetColumn("create_time", time.Now())
	scope.SetColumn("timestamp", time.Now().Unix())
	scope.SetColumn("update_time", time.Now())
	scope.SetColumn("is_valid", true)
	return
}

func BeforeUpdate(scope *gorm.Scope) {
	if scope.HasColumn("update_time") {
		scope.SetColumn("update_time", time.Now().Local())
	}
}

// apis
func NewDevice(c *gin.Context) {
	var (
		enterprise_id string
		uuid          string
		device_name   string
	)
	enterprise_id = c.PostForm("enterprise_id")
	uuid = c.PostForm("uuid")
	device_name = c.DefaultPostForm("device_name", "考勤机1")

	db.Create(&Device{
		EnterpriseID: enterprise_id,
		UUID:         uuid,
		DeviceName:   device_name,
		VoiceLevel:   1,
	})
	c.JSON(Succeed(""))
}

func DeviceList(c *gin.Context) {
	var devices []Device
	//devices = append(devices, Device{})
	db.Find(&devices, "is_valid = ?", true)
	//time := time.Now()
	for i := 0; i < len(devices); i++ {
		data, _ := cache.Get(fmt.Sprintf("device-%v-status", devices[i].ID)).Result()
		//status := make(map[string]string)
		var status OnLineStatus
		json.Unmarshal([]byte(data), &status)
		if status.Flag == "" {
			status.Flag = "0"
		}
		//devices[i].Status = OnLineStatus{status["flag"], status["last_activity_time"]} //
		devices[i].Status = status
	}
	c.JSON(Succeed(devices))
}

func DeleteDevice(c *gin.Context) {
	var device Device
	object_id := c.Query("object_id")
	status := c.DefaultQuery("status", "0")
	//db.Model(&device).Where("object_id = ?", object_id).Update("is_valid", false)
	db.Where("object_id = ?", object_id).First(&device)
	if device.ID != "" {
		device.IsVaild = status == "1"
		db.Save(&device)
	}
	c.JSON(Succeed(""))
}

func ActiveDevice(c *gin.Context) {
	object_id := c.PostForm("object_id")
	v, _ := json.Marshal(OnLineStatus{"1", time.Now().Format("2006-01-02 15:04:05")})
	cache.Set(fmt.Sprintf("device-%v-status", object_id), string(v), time.Second*60)
	// fmt.Printf("%v", cache.Get(fmt.Sprintf("device-%v-status", object_id)))
	c.JSON(Succeed(""))
}

// todo 已解绑的设备列表 用于恢复绑定
func ValidDeviceList(c *gin.Context) {
	var devices []Device
	// db.Select("select * from device where is_valid = ? distinct enterprise_id", false).Find(&devices)
	c.JSON(Succeed(devices))
}

// global
var db *gorm.DB
var cache *redis.Client

func main() {

	db = InitDB()
	cache = InitRedis()

	defer db.Close()
	defer cache.Close()
	db.SingularTable(true)

	db.Table("testdevice").AutoMigrate(&Device{})

	db.Callback().Update().Before("gorm:update").Register("my_plugin:before_update", BeforeUpdate)
	db.Callback().Create().Before("gorm:create").Register("my_plugin:before_create", BeforeCreate)

	router := gin.Default()
	router.Use(Cors())
	api_admin := router.Group("/admin")
	{
		api_admin.POST("/device", NewDevice)
		api_admin.GET("/device", DeviceList)
		api_admin.DELETE("/device", DeleteDevice)
		api_admin.POST("/device/active", ActiveDevice)
		api_admin.GET("/device/valid_list", ValidDeviceList)
	}
	err := router.Run("0.0.0.0:8080") // listen and serve on 0.0.0.0:8080
	if err != nil {
		fmt.Printf("server run error")
	}
}
