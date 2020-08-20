package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func passwordRecipeInt(i int) *int    { return &i }
func passwordRecipeBool(b bool) *bool { return &b }

func TestPasswordRecipe_Parse(t *testing.T) {
	cases := []struct {
		desc  string
		input string
		PasswordRecipe
	}{
		{
			desc: "empty",
		},
		{
			desc:  "single int",
			input: "length:2",
			PasswordRecipe: PasswordRecipe{
				Length: passwordRecipeInt(2),
			},
		},
		{
			desc:  "single bool",
			input: "noUpper:true",
			PasswordRecipe: PasswordRecipe{
				NoUpper: passwordRecipeBool(true),
			},
		},
		{
			desc:  "override",
			input: "length:2,length:3",
			PasswordRecipe: PasswordRecipe{
				Length: passwordRecipeInt(3),
			},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			pr := &PasswordRecipe{}
			pr.Parse(c.input)
			assert.Equal(t, &c.PasswordRecipe, pr)
		})
	}
}

func TestSecretManagerReference_AsFileSources(t *testing.T) {
	cases := []struct {
		desc     string
		ref      SecretManagerReference
		expected string
	}{
		{
			desc: "empty key",
			ref: SecretManagerReference{
				Project: "foo",
				Secret:  "bar",
			},
			expected: "sm://foo/bar",
		},
		{
			desc: "full reference",
			ref: SecretManagerReference{
				Key:     "test",
				Project: "foo",
				Secret:  "bar",
				Version: "1",
			},
			expected: "test=sm://foo/bar#1",
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			actual, err := secretManagerSecretsAsFileSources([]SecretManagerReference{c.ref})
			if assert.NoError(t, err) {
				assert.Equal(t, c.expected, actual[0])
			}
		})
	}
}
