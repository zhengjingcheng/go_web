package orm

import (
	"database/sql"
	"errors"
	"fmt"
	zjcLog "github.com/zhengjingcheng/zjcgo/log"
	"reflect"
	"strings"
	"time"
)

type ZjcDb struct {
	db     *sql.DB
	logger *zjcLog.Logger
	Prefix string
}
type ZjcSeeion struct {
	db          *ZjcDb
	tableName   string
	fieldName   []string
	placeHolder []string
	values      []any
	updateParam strings.Builder
	whereParam  strings.Builder
	whereValues []any
}

func Open(driverName string, source string) *ZjcDb {
	db, err := sql.Open(driverName, source)

	if err != nil {
		panic(err)
	}
	//最大空闲连接数，默认不配置，是2个最大空闲连接
	db.SetMaxIdleConns(5)
	//最大连接数，默认不配置，是不限制最大连接数
	db.SetMaxOpenConns(100)
	// 连接最大存活时间
	db.SetConnMaxLifetime(time.Minute * 3)
	//空闲连接最大存活时间
	db.SetConnMaxIdleTime(time.Minute * 1)
	zjcDb := &ZjcDb{
		db:     db,
		logger: zjcLog.Default(),
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	return zjcDb
}

func (db *ZjcDb) Close() error {
	return db.db.Close()
}

func (db *ZjcDb) SetMaxIdleConns(n int) {
	db.db.SetMaxIdleConns(5)
}

func (db *ZjcDb) New() *ZjcSeeion {
	return &ZjcSeeion{
		db: db,
	}
}
func (s *ZjcSeeion) Table(name string) *ZjcSeeion {
	s.tableName = name
	return s
}
func (s *ZjcSeeion) Insert(data any) (int64, int64, error) {
	//每个操作是独立的 互不影响的session
	//insert into table (xxx,xxx) values(？.?)
	s.fieldNames(data)
	query := fmt.Sprintf("insert into %s (%s) values (%s)", s.tableName, strings.Join(s.fieldName, ","), strings.Join(s.placeHolder, ","))
	s.db.logger.Info(query)
	stmt, err := s.db.db.Prepare(query)
	if err != nil {
		return -1, -1, err
	}
	r, err := stmt.Exec(s.values...)
	if err != nil {
		return -1, -1, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return -1, -1, err
	}
	affected, err := r.RowsAffected()
	if err != nil {
		return -1, -1, err
	}
	return id, affected, nil
}
func (s *ZjcSeeion) fieldNames(data any) {
	//反射
	t := reflect.TypeOf(data)
	v := reflect.ValueOf(data)
	//必须传递指针
	if t.Kind() != reflect.Pointer {
		panic(errors.New("data must be pointer"))
	}
	tVar := t.Elem()
	vVar := v.Elem()
	if s.tableName == "" {
		//如果没有表名，则给出一个默认的表名
		s.tableName = s.db.Prefix + strings.ToLower(Name(tVar.Name()))
	}
	for i := 0; i < tVar.NumField(); i++ {
		fieldName := tVar.Field(i).Name
		tag := tVar.Field(i).Tag //把tag提取出来
		sqlTag := tag.Get("zjcorm")
		if sqlTag == "" {
			//如果用户没给tag字节写个默认值
			sqlTag = strings.ToLower(Name(fieldName)) //转成小写
		} else {
			if strings.Contains(sqlTag, "auto_increment") {
				//自增长的主键id
				continue
			}
			if strings.Contains(sqlTag, ",") {
				//如果Tag里包含逗号，    id,name,则取出第一个tag
				sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
			}
			id := vVar.Field(i).Interface()
			if sqlTag == "id" && IsAutoId(id) {
				//如果是自增长的Id就不处理
				continue
			}
		}
		//把字段名称加进去
		s.fieldName = append(s.fieldName, sqlTag)
		//添加占位符
		s.placeHolder = append(s.placeHolder, "?")
		//添加值
		s.values = append(s.values, vVar.Field(i).Interface())
		fmt.Println(vVar.Field(i).Interface())
	}

}

//更新
func (s *ZjcSeeion) Update(data ...any) (int64, int64, error) {
	//Update("age",1) or Update(user)
	if len(data) == 0 || len(data) > 2 {
		return -1, -1, errors.New("param not valid")
	}
	single := true
	if len(data) == 2 {
		single = false
	}
	//写成格式：update table set age=?,name=? where id=?
	if !single {
		//如果前边有值需要加一个逗号
		if s.updateParam.String() != "" {
			s.updateParam.WriteString(",")
		}
		s.updateParam.WriteString(data[0].(string))
		s.updateParam.WriteString(" = ? ")
		s.values = append(s.values, data[1])
	}
	query := fmt.Sprintf("update %s set %s", s.tableName, s.updateParam.String())
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	s.db.logger.Info(sb.String())
	stmt, err := s.db.db.Prepare(sb.String())
	if err != nil {
		return -1, -1, err
	}
	s.values = append(s.values, s.whereValues...)
	r, err := stmt.Exec(s.values...)
	if err != nil {
		return -1, -1, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return -1, -1, err
	}
	affected, err := r.RowsAffected()
	if err != nil {
		return -1, -1, err
	}
	return id, affected, nil
}
func (s *ZjcSeeion) Where(field string, value any) *ZjcSeeion {
	//id = 1
	if s.whereParam.String() == "" {
		s.whereParam.WriteString("where ")
	} else {
		s.whereParam.WriteString(", ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" = ")
	s.whereParam.WriteString(" ? ")
	s.whereValues = append(s.whereValues, value)
	return s
}
func (s *ZjcSeeion) InsertBash(data []any) (int64, int64, error) {
	//insert into table (xxx,xxx) values(？.?),(?,?)
	if len(data) == 0 {
		return -1, -1, errors.New("no data insert") //没有数据
	}

	s.fieldNames(data[0])
	query := fmt.Sprintf("insert into %s (%s) values ", s.tableName, strings.Join(s.fieldName, ","))

	var sb strings.Builder
	sb.WriteString(query)
	for index, _ := range data {
		sb.WriteString("(")
		sb.WriteString(strings.Join(s.placeHolder, ","))
		sb.WriteString(")")
		if index < len(data)-1 {
			sb.WriteString(",")
		}
	}
	s.batchValues(data)
	s.db.logger.Info(sb.String())
	stmt, err := s.db.db.Prepare(sb.String())
	if err != nil {
		return -1, -1, err
	}
	r, err := stmt.Exec(s.values...)
	if err != nil {
		return -1, -1, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return -1, -1, err
	}
	affected, err := r.RowsAffected()
	if err != nil {
		return -1, -1, err
	}
	return id, affected, nil
}

func (s *ZjcSeeion) batchValues(data []any) {

	s.values = make([]any, 0)
	for _, v := range data {
		//反射
		t := reflect.TypeOf(v)
		v := reflect.ValueOf(v)
		//必须传递指针
		if t.Kind() != reflect.Pointer {
			panic(errors.New("data must be pointer"))
		}
		tVar := t.Elem()
		vVar := v.Elem()
		for i := 0; i < tVar.NumField(); i++ {
			fieldName := tVar.Field(i).Name
			tag := tVar.Field(i).Tag //把tag提取出来
			sqlTag := tag.Get("zjcorm")
			if sqlTag == "" {
				//如果用户没给tag字节写个默认值
				sqlTag = strings.ToLower(Name(fieldName)) //转成小写
			} else {
				if strings.Contains(sqlTag, "auto_increment") {
					//自增长的主键id
					continue
				}
			}
			id := vVar.Field(i).Interface()
			if strings.ToLower(sqlTag) == "id" && IsAutoId(id) {
				//如果是自增长的Id就不处理
				continue
			}
			//添加值
			s.values = append(s.values, vVar.Field(i).Interface())
		}
	}
}

func IsAutoId(id any) bool {
	t := reflect.TypeOf(id)
	switch t.Kind() {
	case reflect.Int64:
		if id.(int64) <= 0 {
			return true
		}
	case reflect.Int32:
		if id.(int32) <= 0 {
			return true
		}
	case reflect.Int:
		if id.(int) <= 0 {
			return true
		}
	default:
		return false
	}
	return false
}

//处理fileName
func Name(name string) string {
	var names = name[:]
	lastIndex := 0
	var sb strings.Builder
	for index, value := range names {
		if value >= 65 && value <= 90 {
			//大写字母
			if index == 0 {
				//首字母大写不用考虑
				continue
			}
			sb.WriteString(name[:index])
			sb.WriteString("_")
			lastIndex = index
		}
	}
	sb.WriteString(name[lastIndex:])
	return sb.String()
}
