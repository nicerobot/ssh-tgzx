package app

type (
	envName string // envName represents an environment variable name.
)

// envPrefix is a type for environment variable prefixes.
type envPrefix string

// Env constructs an environment variable name with the app prefix.
func (e envPrefix) Env(name envName) envName {
	return envName(string(e) + string(name))
}
