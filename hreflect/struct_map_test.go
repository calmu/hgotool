package hreflect

import (
	"testing"
)

// TestStructToMap 测试结构体到map的转换
func TestStructToMap(t *testing.T) {
	type Person struct {
		Name    string `json:"name"`
		Age     int    `json:"age"`
		Address string `json:"address"`
		Email   string // 没有json标签
	}

	person := Person{
		Name:    "John Doe",
		Age:     30,
		Address: "New York",
		Email:   "john@example.com",
	}

	result, err := StructToMap(person)
	if err != nil {
		t.Fatalf("StructToMap failed: %v", err)
	}

	if len(result) != 4 {
		t.Errorf("Expected 4 fields, got %d", len(result))
	}

	if name, ok := result["name"]; !ok || name != "John Doe" {
		t.Errorf("Expected name to be 'John Doe', got %v", name)
	}

	if age, ok := result["age"]; !ok || age != 30 {
		t.Errorf("Expected age to be 30, got %v", age)
	}

	if email, ok := result["Email"]; !ok || email != "john@example.com" {
		t.Errorf("Expected Email to be 'john@example.com', got %v", email)
	}
}

// TestMapToStruct 测试map到结构体的转换
func TestMapToStruct(t *testing.T) {
	type Person struct {
		Name    string `json:"name"`
		Age     int    `json:"age"`
		Address string `json:"address"`
		Email   string // 没有json标签
	}

	data := map[string]interface{}{
		"name":    "Jane Doe",
		"age":     25,
		"address": "Los Angeles",
		"Email":   "jane@example.com",
	}

	var person Person
	err := MapToStruct(data, &person)
	if err != nil {
		t.Fatalf("MapToStruct failed: %v", err)
	}

	if person.Name != "Jane Doe" {
		t.Errorf("Expected Name to be 'Jane Doe', got %v", person.Name)
	}

	if person.Age != 25 {
		t.Errorf("Expected Age to be 25, got %v", person.Age)
	}

	if person.Address != "Los Angeles" {
		t.Errorf("Expected Address to be 'Los Angeles', got %v", person.Address)
	}

	if person.Email != "jane@example.com" {
		t.Errorf("Expected Email to be 'jane@example.com', got %v", person.Email)
	}
}

// TestStructToMapWithTag 测试带有json标签的结构体转换
func TestStructToMapWithTag(t *testing.T) {
	type User struct {
		ID       int    `json:"id"`
		FullName string `json:"full_name"`
		IsActive bool   `json:"is_active"`
		Score    float64
	}

	user := User{
		ID:       1,
		FullName: "Alice Smith",
		IsActive: true,
		Score:    95.5,
	}

	result, err := StructToMap(user)
	if err != nil {
		t.Fatalf("StructToMap failed: %v", err)
	}

	if id, ok := result["id"]; !ok || id != 1 {
		t.Errorf("Expected id to be 1, got %v", id)
	}

	if fullName, ok := result["full_name"]; !ok || fullName != "Alice Smith" {
		t.Errorf("Expected full_name to be 'Alice Smith', got %v", fullName)
	}

	if isActive, ok := result["is_active"]; !ok || isActive != true {
		t.Errorf("Expected is_active to be true, got %v", isActive)
	}

	if score, ok := result["Score"]; !ok || score != 95.5 {
		t.Errorf("Expected Score to be 95.5, got %v", score)
	}
}

// TestMapToStructWithTag 测试map到带标签结构体的转换
func TestMapToStructWithTag(t *testing.T) {
	type User struct {
		ID       int    `json:"id"`
		FullName string `json:"full_name"`
		IsActive bool   `json:"is_active"`
		Score    float64
	}

	data := map[string]interface{}{
		"id":        2,
		"full_name": "Bob Johnson",
		"is_active": false,
		"Score":     87.2,
	}

	var user User
	err := MapToStruct(data, &user)
	if err != nil {
		t.Fatalf("MapToStruct failed: %v", err)
	}

	if user.ID != 2 {
		t.Errorf("Expected ID to be 2, got %v", user.ID)
	}

	if user.FullName != "Bob Johnson" {
		t.Errorf("Expected FullName to be 'Bob Johnson', got %v", user.FullName)
	}

	if user.IsActive != false {
		t.Errorf("Expected IsActive to be false, got %v", user.IsActive)
	}

	if user.Score != 87.2 {
		t.Errorf("Expected Score to be 87.2, got %v", user.Score)
	}
}

// TestStructToMapWithNested 测试嵌套结构体转换
func TestStructToMapWithNested(t *testing.T) {
	type Address struct {
		City    string `json:"city"`
		Country string `json:"country"`
	}

	type Person struct {
		Name    string  `json:"name"`
		Age     int     `json:"age"`
		Address Address `json:"address"`
	}

	address := Address{
		City:    "Beijing",
		Country: "China",
	}

	person := Person{
		Name:    "Zhang San",
		Age:     28,
		Address: address,
	}

	result, err := StructToMap(person)
	if err != nil {
		t.Fatalf("StructToMap failed: %v", err)
	}

	if name, ok := result["name"]; !ok || name != "Zhang San" {
		t.Errorf("Expected name to be 'Zhang San', got %v", name)
	}

	if addr, ok := result["address"]; ok {
		if addrObj, ok := addr.(Address); ok {
			if addrObj.City != "Beijing" {
				t.Errorf("Expected address city to be 'Beijing', got %v", addrObj.City)
			}
		} else {
			t.Errorf("Expected address to be Address type")
		}
	} else {
		t.Errorf("Expected address field to exist")
	}
}

// TestMapToStructWithNested 测试嵌套结构体转换
func TestMapToStructWithNested(t *testing.T) {
	type Address struct {
		City    string `json:"city"`
		Country string `json:"country"`
	}

	type Person struct {
		Name    string  `json:"name"`
		Age     int     `json:"age"`
		Address Address `json:"address"`
	}

	addrData := map[string]interface{}{
		"city":    "Shanghai",
		"country": "China",
	}

	data := map[string]interface{}{
		"name":    "Li Si",
		"age":     32,
		"address": addrData,
	}

	var person Person
	err := MapToStruct(data, &person)
	if err != nil {
		t.Fatalf("MapToStruct failed: %v", err)
	}

	if person.Name != "Li Si" {
		t.Errorf("Expected Name to be 'Li Si', got %v", person.Name)
	}

	if person.Address.City != "Shanghai" {
		t.Errorf("Expected Address.City to be 'Shanghai', got %v", person.Address.City)
	}

	if person.Address.Country != "China" {
		t.Errorf("Expected Address.Country to be 'China', got %v", person.Address.Country)
	}
}

// TestMapToStructWithTypeConversion 测试类型转换功能
func TestMapToStructWithTypeConversion(t *testing.T) {
	type Config struct {
		Port    int     `json:"port"`
		Host    string  `json:"host"`
		Enabled bool    `json:"enabled"`
		Timeout float64 `json:"timeout"`
	}

	data := map[string]interface{}{
		"port":    "8080", // string -> int
		"host":    "localhost",
		"enabled": "true", // string -> bool
		"timeout": "30.5", // string -> float64
	}

	var config Config
	err := MapToStruct(data, &config)
	if err != nil {
		t.Fatalf("MapToStruct failed: %v", err)
	}

	if config.Port != 8080 {
		t.Errorf("Expected Port to be 8080, got %v", config.Port)
	}

	if config.Host != "localhost" {
		t.Errorf("Expected Host to be 'localhost', got %v", config.Host)
	}

	if !config.Enabled {
		t.Errorf("Expected Enabled to be true, got %v", config.Enabled)
	}

	if config.Timeout != 30.5 {
		t.Errorf("Expected Timeout to be 30.5, got %v", config.Timeout)
	}
}

// TestStructToMapPointer 测试指针结构体转换
func TestStructToMapPointer(t *testing.T) {
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	person := &Person{
		Name: "Wang Wu",
		Age:  29,
	}

	result, err := StructToMap(person)
	if err != nil {
		t.Fatalf("StructToMap failed: %v", err)
	}

	if name, ok := result["name"]; !ok || name != "Wang Wu" {
		t.Errorf("Expected name to be 'Wang Wu', got %v", name)
	}

	if age, ok := result["age"]; !ok || age != 29 {
		t.Errorf("Expected age to be 29, got %v", age)
	}
}

// TestMapToStructWithSlice 测试切片字段的转换
func TestMapToStructWithSlice(t *testing.T) {
	type Student struct {
		Name     string   `json:"name"`
		Grades   []int    `json:"grades"`
		Subjects []string `json:"subjects"`
	}

	data := map[string]interface{}{
		"name":     "Student A",
		"grades":   []interface{}{90, 85, 95},
		"subjects": []interface{}{"Math", "Science", "English"},
	}

	var student Student
	err := MapToStruct(data, &student)
	if err != nil {
		t.Fatalf("MapToStruct failed: %v", err)
	}

	if student.Name != "Student A" {
		t.Errorf("Expected Name to be 'Student A', got %v", student.Name)
	}

	if len(student.Grades) != 3 || student.Grades[0] != 90 {
		t.Errorf("Expected grades to be [90, 85, 95], got %v", student.Grades)
	}

	if len(student.Subjects) != 3 || student.Subjects[0] != "Math" {
		t.Errorf("Expected subjects to be ['Math', 'Science', 'English'], got %v", student.Subjects)
	}
}

// TestComplexExample 综合示例
func TestComplexExample(t *testing.T) {
	type Contact struct {
		Phone string `json:"phone"`
		Email string `json:"email"`
	}

	type Profile struct {
		Name    string   `json:"name"`
		Age     int      `json:"age"`
		Contact Contact  `json:"contact"`
		Tags    []string `json:"tags"`
		Active  bool     `json:"active"`
	}

	contactData := map[string]interface{}{
		"phone": "123-456-7890",
		"email": "test@example.com",
	}

	data := map[string]interface{}{
		"name":    "Complex User",
		"age":     "35", // string -> int conversion
		"contact": contactData,
		"tags":    []interface{}{"developer", "gopher"},
		"active":  "true", // string -> bool conversion
	}

	var profile Profile
	err := MapToStruct(data, &profile)
	if err != nil {
		t.Fatalf("MapToStruct failed: %v", err)
	}

	if profile.Name != "Complex User" {
		t.Errorf("Expected Name to be 'Complex User', got %v", profile.Name)
	}

	if profile.Age != 35 {
		t.Errorf("Expected Age to be 35, got %v", profile.Age)
	}

	if profile.Contact.Phone != "123-456-7890" {
		t.Errorf("Expected Contact.Phone to be '123-456-7890', got %v", profile.Contact.Phone)
	}

	if len(profile.Tags) != 2 || profile.Tags[0] != "developer" {
		t.Errorf("Expected Tags to be ['developer', 'gopher'], got %v", profile.Tags)
	}

	if !profile.Active {
		t.Errorf("Expected Active to be true, got %v", profile.Active)
	}

	// 将结构体转换回map
	result, err := StructToMap(profile)
	if err != nil {
		t.Fatalf("StructToMap failed: %v", err)
	}

	if name, ok := result["name"]; !ok || name != "Complex User" {
		t.Errorf("Expected name to be 'Complex User', got %v", name)
	}
}
