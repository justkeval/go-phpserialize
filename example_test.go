package phpserialize_test

import (
	"fmt"

	"phpserialize"
)

func ExampleMarshal() {
	type Product struct {
		Name  string   `php:"name"`
		Price float64  `php:"price"`
		Tags  []string `php:"tags"`
	}
	b, err := phpserialize.Marshal(Product{
		Name:  "Book",
		Price: 9.99,
		Tags:  []string{"paper", "gift"},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
	// Output: a:3:{s:4:"name";s:4:"Book";s:5:"price";d:9.99;s:4:"tags";a:2:{i:0;s:5:"paper";i:1;s:4:"gift";}}
}

func ExampleUnmarshal() {
	type User struct {
		Name string `php:"name"`
		Age  int    `php:"age"`
	}
	data := []byte(`a:2:{s:4:"name";s:3:"Ada";s:3:"age";i:36;}`)
	var u User
	if err := phpserialize.Unmarshal(data, &u); err != nil {
		panic(err)
	}
	fmt.Printf("%s is %d\n", u.Name, u.Age)
	// Output: Ada is 36
}

func ExampleUnmarshal_interface() {
	// Decoding into an empty interface uses the default native mapping.
	data := []byte("a:2:{i:0;i:10;i:1;i:20;}")
	var v any
	if err := phpserialize.Unmarshal(data, &v); err != nil {
		panic(err)
	}
	fmt.Printf("%#v\n", v)
	// Output: []interface {}{10, 20}
}

func ExampleParse() {
	// Parse exposes the raw structure, including object class names, which
	// Unmarshal discards.
	v, err := phpserialize.Parse([]byte(`O:4:"User":1:{s:4:"name";s:3:"Bob";}`))
	if err != nil {
		panic(err)
	}
	fmt.Println(v.Object.ClassName)
	// Output: User
}
