package resolver

import "testing"

func TestDefaultConverterName_DifferentTypeNames(t *testing.T) {
	got := DefaultConverterName("example.com/src", "User", "example.com/dst", "UserResponse")
	if got != "ConvertUserToUserResponse" {
		t.Fatalf("unexpected func name: %s", got)
	}
}

func TestDefaultConverterName_SameTypeNameUsesPackageToken(t *testing.T) {
	got := DefaultConverterName("example.com/domain-model", "User", "example.com/dto", "User")
	if got != "ConvertDomainModelUserToDtoUser" {
		t.Fatalf("unexpected func name: %s", got)
	}
}
