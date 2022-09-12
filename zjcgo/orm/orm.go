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
	tx          *sql.Tx //事务
	beginTx     bool    //是否开启事务
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

func (db *ZjcDb) New(data any) *ZjcSeeion {
	m := &ZjcSeeion{
		db: db,
	}
	t := reflect.TypeOf(data)
	//必须传递指针
	if t.Kind() != reflect.Pointer {
		panic(errors.New("data must be pointer"))
	}
	tVar := t.Elem()
	if m.tableName == "" {
		//如果没有表名，则给出一个默认的表名
		m.tableName = m.db.Prefix + strings.ToLower(Name(tVar.Name()))
	}
	return m
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
	var stmt *sql.Stmt
	var err error
	if s.beginTx {
		stmt, err = s.tx.Prepare(query)
	} else {
		stmt, err = s.db.db.Prepare(query)
	}
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
	s.tx.Commit()
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
	} else {
		updateData := data[0]
		t := reflect.TypeOf(updateData)
		v := reflect.ValueOf(updateData)
		//必须传递指针
		if t.Kind() != reflect.Pointer {
			panic(errors.New("updateData must be pointer"))
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
			if s.updateParam.String() != "" {
				s.updateParam.WriteString(",")
			}
			s.updateParam.WriteString(sqlTag)
			s.updateParam.WriteString(" = ? ")
			s.values = append(s.values, vVar.Field(i).Interface())
		}
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

func (s *ZjcSeeion) Count() (int64, error) {
	query := fmt.Sprintf("select count(*) from %s ", s.tableName)
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	s.db.logger.Info(sb.String())
	stmt, err := s.db.db.Prepare(sb.String())
	if err != nil {
		return 0, err
	}
	row := stmt.QueryRow(s.whereValues...)
	if row.Err() != nil {
		return 0, err
	}
	var result int64
	err = row.Scan(&result)
	if err != nil {
		return 0, err
	}
	return result, nil
}

func (s *ZjcSeeion) Where(field string, value any) *ZjcSeeion {
	//id = 1
	if s.whereParam.String() == "" {
		s.whereParam.WriteString("where ")
	} else {
		s.whereParam.WriteString(" and ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" = ")
	s.whereParam.WriteString(" ? ")
	s.whereValues = append(s.whereValues, value)
	return s
}

//其他查询条件
func (s *ZjcSeeion) Like(field string, value any) *ZjcSeeion {
	//name like %s%
	if s.whereParam.String() == "" {
		s.whereParam.WriteString("where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" like ")
	s.whereParam.WriteString(" ? ")
	s.whereValues = append(s.whereValues, "%"+value.(string)+"%")
	return s
}
func (s *ZjcSeeion) Group(field ...string) *ZjcSeeion {
	//group by aa,bb
	s.whereParam.WriteString(" group by ")
	s.whereParam.WriteString(strings.Join(field, ","))
	return s
}
func (s *ZjcSeeion) OrderDesc(field ...string) *ZjcSeeion {
	//Order by aa,bb desc
	s.whereParam.WriteString(" order by ")
	s.whereParam.WriteString(strings.Join(field, ","))
	s.whereParam.WriteString(" desc ")
	return s
}
func (s *ZjcSeeion) OrderAsc(field ...string) *ZjcSeeion {
	//Order by aa,bb asc
	s.whereParam.WriteString(" order by ")
	s.whereParam.WriteString(strings.Join(field, ","))
	s.whereParam.WriteString(" asc ")
	return s
}

//Order Order("aa","desc","bb","asc")
func (s *ZjcSeeion) Order(field ...string) *ZjcSeeion {
	//Order by aa desc,bb asc
	if len(field)%2 != 0 {
		panic("field num not true")
	}
	s.whereParam.WriteString(" order by ")
	for index, v := range field {
		s.whereParam.WriteString(v + " ")
		if index%2 != 0 && index < len(field)-1 {
			s.whereParam.WriteString(",")
		}
	}
	return s
}

func (s *ZjcSeeion) Delete() (int64, error) {
	//delete from table where id = ?
	query := fmt.Sprintf("delete from %s ", s.tableName)
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	s.db.logger.Info(sb.String())
	stmt, err := s.db.db.Prepare(sb.String())
	if err != nil {
		return 0, err
	}
	r, err := stmt.Exec(s.whereValues...)
	if err != nil {
		return 0, err
	}
	return r.RowsAffected()
}

func (s *ZjcSeeion) And() *ZjcSeeion {
	s.whereParam.WriteString(" and ")
	return s
}

func (s *ZjcSeeion) Or() *ZjcSeeion {
	s.whereParam.WriteString(" or ")
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

/*
	原生sql的支持
*/
func (s *ZjcSeeion) Exec(sql string, values ...any) (int64, error) {
	stmt, err := s.db.db.Prepare(sql)
	if err != nil {
		return 0, err
	}
	r, err := stmt.Exec(values)
	if err != nil {
		return 0, err
	}
	if strings.Contains(strings.ToLower(sql), "insert") {
		return r.LastInsertId()
	}
	return r.RowsAffected()
}
func (s *ZjcSeeion) QueryRow(sql string, data any, queryValues ...any) error {
	t := reflect.TypeOf(data)
	stmt, err := s.db.db.Prepare(sql)
	if err != nil {
		return err
	}
	rows, err := stmt.Query(queryValues...)
	if err != nil {
		return err
	}
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	values := make([]any, len(columns))
	fieldScan := make([]any, len(columns))
	for i := range fieldScan {
		fieldScan[i] = &values[i]
	}
	if rows.Next() {
		err := rows.Scan(fieldScan...)
		if err != nil {
			return err
		}
		tVar := t.Elem()
		vVar := reflect.ValueOf(data).Elem()
		for i := 0; i < tVar.NumField(); i++ {
			name := tVar.Field(i).Name
			tag := tVar.Field(i).Tag
			//id,auto
			sqlTag := tag.Get("zjcorm")
			if sqlTag == "" {
				sqlTag = strings.ToLower(Name(name))
			} else {
				if strings.Contains(sqlTag, ",") {
					sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
				}
			}
			for j, colName := range columns {
				if sqlTag == colName {
					target := values[j]
					targetValue := reflect.ValueOf(target)
					fieldType := tVar.Field(i).Type
					//这样不行 类型不匹配 转换类型
					result := reflect.ValueOf(targetValue.Interface()).Convert(fieldType)
					vVar.Field(i).Set(result)
				}
			}
		}
	}
	return nil
}

/*
	查询语句
*/
//select * from table where id =1000 难点：
//查询一个
func (s *ZjcSeeion) SelectOne(data any, fields ...string) error {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Pointer {
		return errors.New("data must be pointer")
	}
	fieldStr := "*"
	if len(fields) > 0 {
		fieldStr = strings.Join(fields, ",")
	}
	query := fmt.Sprintf("select %s from %s ", fieldStr, s.tableName)
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	s.db.logger.Info(sb.String())

	stmt, err := s.db.db.Prepare(sb.String())
	if err != nil {
		return err
	}
	rows, err := stmt.Query(s.whereValues...)
	if err != nil {
		return err
	}
	//id user_name age
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	values := make([]any, len(columns))
	fieldScan := make([]any, len(columns))
	for i := range fieldScan {
		fieldScan[i] = &values[i]
	}
	if rows.Next() {
		err := rows.Scan(fieldScan...)
		if err != nil {
			return err
		}
		tVar := t.Elem()
		vVar := reflect.ValueOf(data).Elem()
		for i := 0; i < tVar.NumField(); i++ {
			name := tVar.Field(i).Name
			tag := tVar.Field(i).Tag
			//id,auto
			sqlTag := tag.Get("zjcorm")
			if sqlTag == "" {
				sqlTag = strings.ToLower(Name(name))
			} else {
				if strings.Contains(sqlTag, ",") {
					sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
				}
			}
			for j, colName := range columns {
				if sqlTag == colName {
					target := values[j]
					targetValue := reflect.ValueOf(target)
					fieldType := tVar.Field(i).Type
					//这样不行 类型不匹配 转换类型
					result := reflect.ValueOf(targetValue.Interface()).Convert(fieldType)
					vVar.Field(i).Set(result)
				}
			}
		}
	}
	return nil
}

/*
	查询多行
*/
func (s *ZjcSeeion) Select(data any, fields ...string) ([]any, error) {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Pointer {
		return nil, errors.New("data must be pointer")
	}
	fieldStr := "*"
	if len(fields) > 0 {
		fieldStr = strings.Join(fields, ",")
	}
	query := fmt.Sprintf("select %s from %s ", fieldStr, s.tableName)
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	s.db.logger.Info(sb.String())

	stmt, err := s.db.db.Prepare(sb.String())
	if err != nil {
		return nil, err
	}
	rows, err := stmt.Query(s.whereValues...)
	if err != nil {
		return nil, err
	}
	//id user_name age
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := make([]any, 0)
	for {
		if rows.Next() {
			//由于 传进来的是一个指针地址  如果每次赋值 实际都是一个 result里面值都一样
			//每次查询的时候 data都换一个地址
			data := reflect.New(t.Elem()).Interface()
			values := make([]any, len(columns))
			fieldScan := make([]any, len(columns))
			for i := range fieldScan {
				fieldScan[i] = &values[i]
			}
			err := rows.Scan(fieldScan...)
			if err != nil {
				return nil, err
			}
			tVar := t.Elem()
			vVar := reflect.ValueOf(data).Elem()
			for i := 0; i < tVar.NumField(); i++ {
				name := tVar.Field(i).Name
				tag := tVar.Field(i).Tag
				//id,auto
				sqlTag := tag.Get("zjcorm")
				if sqlTag == "" {
					sqlTag = strings.ToLower(Name(name))
				} else {
					if strings.Contains(sqlTag, ",") {
						sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
					}
				}
				for j, colName := range columns {
					if sqlTag == colName {
						target := values[j]
						targetValue := reflect.ValueOf(target)
						fieldType := tVar.Field(i).Type
						//这样不行 类型不匹配 转换类型
						result := reflect.ValueOf(targetValue.Interface()).Convert(fieldType)
						vVar.Field(i).Set(result)
					}
				}
			}
			result = append(result, data)
		} else {
			break
		}
	}
	return result, nil
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

/*
   事务
*/
func (s *ZjcSeeion) Begin() error {
	tx, err := s.db.db.Begin()
	if err != nil {
		return err
	}
	s.tx = tx
	s.beginTx = true
	return nil
}
func (s *ZjcSeeion) Commit() error {
	err := s.tx.Commit()
	if err != nil {
		return err
	}
	s.beginTx = false
	return nil
}
func (s *ZjcSeeion) Rollback() error {
	err := s.tx.Rollback()
	if err != nil {
		return err
	}
	s.beginTx = false
	return nil
}