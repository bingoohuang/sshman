package config

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/bingoohuang/sshman/model"
	"github.com/garyburd/redigo/redis"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"log"
	"strconv"
	"strings"
	"time"
)

//订制配置文件解析载体
type conf struct {
	Web      *webset
	Database *database
	Redis    *redisConf
	AliSms   *aliSms
	Jwt      *jwtInfo
}

type jwtInfo struct {
	Key string
}

type aliSms struct {
	Accessid  string
	Accesskey string
	Signname  string
	Template  string
}

type redisConf struct {
	Addr     string
	Password string
}

type database struct {
	Dsn string
}

type webset struct {
	Port string
}

var (
	Conf  = new(conf)
	DB    func() *gorm.DB
	Cache *redis.Pool
)

func LoadConfig(configFilePath string) {
	_, err := toml.DecodeFile(configFilePath, Conf)
	if err != nil {
		log.Panic(err.Error())
	}
	DB = Conf.Database.newDb
	Cache = Conf.Redis.poolInitRedis()
}

func (d *database) newDb() *gorm.DB {
	dsn, err := ParseDataSourceFlag(d.Dsn)
	if err != nil {
		log.Panicf("ParseDataSourceFlag %s err: %v", d.Dsn, err)
	}

	db, err := gorm.Open("mysql",
		fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			dsn.Username, dsn.Password, dsn.Host, OrInt(dsn.Port, 3306), dsn.Database))
	if err != nil {
		log.Panicf("db open err :%s", err.Error())
	}
	if !db.HasTable(&model.User{}) {
		log.Println("init table")
		db.CreateTable(&model.Server{}, &model.User{})
	}

	return db
}

func OrInt(a, b int) int {
	if a == 0 {
		return b
	}

	return a
}

func (r redisConf) poolInitRedis() *redis.Pool {
	return &redis.Pool{
		MaxIdle:     2, //空闲数
		IdleTimeout: 240 * time.Second,
		MaxActive:   1, //最大数
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", r.Addr)
			if err != nil {
				return nil, err
			}
			if r.Password != "" {
				if _, err := c.Do("AUTH", r.Password); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

type DataSourceConfig struct {
	Username, Password, Host string
	Port                     int
	Database                 string
}

// ParseDataSourceFlag parses format like user:pass@host:port/db.
func ParseDataSourceFlag(source string) (dc *DataSourceConfig, err error) {
	dc = &DataSourceConfig{}
	atPos := strings.LastIndex(source, "@")
	if atPos < 0 {
		err = fmt.Errorf("invalid source: %s, should be username:password@host:port", source)
		return
	}

	userPart := source[:atPos]
	if n := strings.LastIndex(userPart, ":"); n > 0 {
		if dc.Username = userPart[:n]; n+1 < len(userPart) {
			dc.Password = userPart[n+1:]
		}
	} else {
		dc.Username = userPart
	}

	if atPos+1 >= len(source) {
		err = fmt.Errorf("invalid source: %s, should be username:password@host:port", source)
		return
	}

	hostPart := source[atPos+1:]

	if n := strings.LastIndex(hostPart, "/"); n > 0 {
		dc.Database = hostPart[n+1:]
		hostPart = hostPart[:n]
	}

	if n := strings.LastIndex(hostPart, ":"); n > 0 {
		if dc.Host = hostPart[:n]; n+1 < len(hostPart) {
			dc.Port, err = strconv.Atoi(hostPart[n+1:])
			if err != nil {
				err = fmt.Errorf("port %s is not a number", hostPart[n+1:])
				return
			}
		}
	} else {
		dc.Host = hostPart
	}

	return
}
