package mysql

import (
	"testing"

	"github.com/JREAMLU/j-core/constant"
	"github.com/jinzhu/gorm"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	consulAddr = "10.200.202.35:8500"
	cronTable  = "Cron"
)

func TestLoadConfig(t *testing.T) {
	Convey("load mysql test", t, func() {
		Convey("load by name", func() {
			dbs, err := Load(consulAddr, "BGCrawler")
			So(err, ShouldBeNil)
			So(len(dbs), ShouldBeGreaterThan, 0)
			for _, db := range dbs {
				err := db.Close()
				So(err, ShouldBeNil)
			}
		})

		Convey("load all", func() {
			dbs, err := Load(consulAddr)
			So(err, ShouldBeNil)
			So(len(dbs), ShouldBeGreaterThan, 0)
			for _, db := range dbs {
				err := db.Close()
				So(err, ShouldBeNil)
			}
		})
	})
}

func TestSQL(t *testing.T) {
	load()
	Convey("sql test", t, func() {
		Convey("insert", func() {
			cron := Cron{
				Name: "jream",
			}
			id, err := insert(cron)
			So(err, ShouldBeNil)
			So(id, ShouldBeGreaterThan, constant.ZeroInt64)
		})

		Convey("query", func() {
			crons, err := query([]int64{1})
			So(err, ShouldBeNil)
			So(len(crons), ShouldBeGreaterThan, 0)
		})

		Convey("raw query", func() {
			crons, err := queryRaw([]int64{1})
			So(err, ShouldBeNil)
			So(len(crons), ShouldBeGreaterThan, 0)
		})
	})
}

var gx map[string]*gorm.DB

func load() {
	var err error
	gx, err = Load(consulAddr, "BGCrawler")
	if err != nil {
		panic(err)
	}
}

func db(isWrite bool) *gorm.DB {
	if isWrite {
		return gx["BGCrawler"]
	}

	return gx[GetReadOnly("BGCrawler")]
}

type Cron struct {
	ID   int64  `gorm:"column:ID;primary_key"`
	Name string `gorm:"column:Name"`
}

func (cron Cron) TableName() string {
	return cronTable
}

func insert(cron Cron) (int64, error) {
	result := db(true).Create(&cron)
	if result.Error != nil {
		return constant.ZeroInt64, result.Error
	}

	return cron.ID, nil
}

func query(ids []int64) (crons []Cron, err error) {
	result := db(false).Where("ID in (?)", ids).Find(&crons)
	if result.Error != nil {
		return crons, result.Error
	}

	return crons, nil
}

func queryRaw(ids []int64) (crons []Cron, err error) {
	sql := `
SELECT  ID, Name
FROM cron
WHERE ID IN (?)
`
	result := db(false).Raw(sql, ids).Scan(&crons)
	if result.Error != nil {
		return crons, result.Error
	}

	return crons, nil
}

/*
CREATE TABLE `Crawler` (
	`ID` INT(11) NOT NULL AUTO_INCREMENT COMMENT 'ID，自增长',
	`Name` varchar(30) NOT NULL DEFAULT '' COMMENT '姓名',
	PRIMARY KEY (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Crawler';
*/
