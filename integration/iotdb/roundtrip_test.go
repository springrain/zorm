package integration

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"testing"
	"time"

	"gitee.com/chunanyong/zorm"
	_ "github.com/apache/iotdb-client-go/v2/database"
)

func TestTreeRoundTrip(t *testing.T) {
	dsn := os.Getenv("IOTDB_TEST_DSN")
	if dsn == "" {
		t.Skip("set IOTDB_TEST_DSN to an IoTDB RPC endpoint")
	}
	dao, err := zorm.NewDBDao(&zorm.DataSourceConfig{DSN: dsn, DriverName: "iotdb", Dialect: "iotdb", DisableTransaction: true})
	if err != nil {
		t.Fatal(err)
	}
	defer dao.CloseDB()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	ctx, err = dao.BindContextDBConnection(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var suffix [12]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		t.Fatal(err)
	}
	database := fmt.Sprintf("root.zorm_smoke_%x", suffix)
	device := database + ".sensor"
	exec := func(s string) error {
		_, err := zorm.UpdateFinder(ctx, zorm.NewFinder().Append(s))
		return err
	}
	if err := exec("CREATE DATABASE " + database); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := exec("DELETE DATABASE " + database); err != nil {
			t.Errorf("dropping %s: %v", database, err)
		} else {
			t.Logf("Dropped %s", database)
		}
	}()
	if err := exec("CREATE TIMESERIES " + device + ".reading WITH DATATYPE=DOUBLE, ENCODING=GORILLA"); err != nil {
		t.Fatal(err)
	}
	var batch []zorm.IEntityMap
	for index, reading := range []float64{21.5, -3.25} {
		row := zorm.NewEntityMap(device)
		row.PkColumnName = ""
		row.Set("timestamp", int64(1700000000123+index))
		row.Set("reading", reading)
		batch = append(batch, row)
	}
	if count, err := zorm.InsertEntityMapSlice(ctx, batch); err != nil || count != 2 {
		t.Fatalf("InsertEntityMapSlice accepted %d rows: %v", count, err)
	}
	rows, err := zorm.QueryMap(ctx, zorm.NewFinder().Append("SELECT reading FROM "+device+" WHERE time >= ?", int64(1700000000123)), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || fmt.Sprint(rows[0][device+".reading"]) != "21.5" || fmt.Sprint(rows[1][device+".reading"]) != "-3.25" {
		t.Fatalf("unexpected readings: %#v", rows)
	}
}
