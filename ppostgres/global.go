package ppostgres

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/EICHI-X/ptools/paerospike"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func GetDbByPSM(ctx context.Context, psm string) (db *gorm.DB, err error) {
	postgresPortValue := os.Getenv("POSTGRES_PORT")
	if postgresPortValue == "" {
		postgresPortValue = "5432"
	}
	p := strings.Split(psm, ".")
	if len(p) != 3 {
		return nil, fmt.Errorf("psm:%v parse err,use like wealth.stock.mainstore", psm)
	}
	l := p[0]
	p[0] = p[2]
	p[2] = l
	postgresWord := strings.Join(p, "_")
	postgresWord = strings.ToUpper(postgresWord)
	user := os.Getenv("POSTGRES_USER_" + postgresWord)
	passwd := os.Getenv("POSTGRES_PASSWD_" + postgresWord)
	dbname := os.Getenv("POSTGRES_DBNAME_" + postgresWord)
	paerospike.NewDefaultClient(psm)
	dnsPsm := strings.Join(p, ".")
	if user == "" || passwd == "" || dbname == "" {
		return nil, fmt.Errorf("psm:%v parse err,use like wealth.stock.mainstore,get use fail", psm)
	}
	dsn := fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=%v ", dnsPsm, user, passwd, dbname, postgresPortValue)
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	return db, err
}
