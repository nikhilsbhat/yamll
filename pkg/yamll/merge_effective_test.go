package yamll_test

import (
	"testing"

	"github.com/nikhilsbhat/yamll/pkg/yamll"
	"github.com/stretchr/testify/assert"
)

func TestYaml_EffectiveMerge(t *testing.T) {
	yamlFile := `---
# Source: internal/fixtures/base3.yaml
#++internal/fixtures/base2.yaml
organizations:
  - thoughtworks
  - google
  - microsoft
---
# Source: internal/fixtures/base.yaml
default: &default
  apiVersion: v1
  kind: ConfigMap
  metadata:
    name: base-config
  data:
    key1: value1
    key2: value2
config1:
  <<: *default
  apiVersion2: v2
config2:
  test: value
---
# Source: internal/fixtures/base2.yaml
names:
  - john doe
  - dexter
---
# Source: http://localhost:3000/database.yaml
mysqldatabase: &mysqldatabase
    hostname: localhost
    port: 3012
    username: root
    password: root
---
# Source: internal/fixtures/import.yaml
#++git+https://github.com/nikhilsbhat/yamll@main?path=internal/fixtures/base2.yaml;{"user_name":"${GIT_USERNAME}","password":"${GITHUB_TOKEN}"}
#++path/to/test.yaml
#++https://test.com/test.yaml;{"user_name":"username","password":"pass","ca_content":"ca_content"}
#++https://test.com/test.yaml;{"user_name":"${username}","password":"${pass}","ca_content":"${ca_content}"}
config2:
  <<: *default
  test: val
config3:
  - <<: *default
  - <<: *mysqldatabase
workflow:
  <<: *mysqldatabase
config1:
  apiVersion2: v4
  apiVersion3: v3
`

	t.Run("", func(t *testing.T) {
		yamlFileContent := yamll.Yaml(yamlFile)

		out, err := yamlFileContent.EffectiveMerge()
		assert.NoError(t, err)
		assert.Nil(t, out)
	})
}
