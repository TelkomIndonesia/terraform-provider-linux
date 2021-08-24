package linux

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testAccProviders map[string]*schema.Provider

var testAccProvider = tfmap{
	attrProviderHost:     `"127.0.0.1"`,
	attrProviderPort:     `22`,
	attrProviderUser:     `"root"`,
	attrProviderPassword: `"root"`,
}

var testAccOverridenProvider = tfmap{
	attrProviderHost:     `"8.8.8.8"`,
	attrProviderPort:     `2222`,
	attrProviderUser:     `"something"`,
	attrProviderPassword: `"else"`,
	attrProviderTimeout:  "1",
}

func testAccInit() {
	testAccProviders = map[string]*schema.Provider{
		"linux": Provider(),
	}

	if f, err := ioutil.ReadFile("testdata/id_rsa"); err == nil {
		testAccProvider.With(attrProviderPrivateKey, string(f))
	}
	if v, ok := os.LookupEnv("TEST_ACC_LINUX_PROVIDER_HOST"); ok {
		testAccProvider.With(attrProviderHost, fmt.Sprintf(`"%s"`, v))
	}
	if v, ok := os.LookupEnv("TEST_ACC_LINUX_PROVIDER_PORT"); ok {
		testAccProvider.With(attrProviderPort, v)
	}
	if v, ok := os.LookupEnv("TEST_ACC_LINUX_PROVIDER_USER"); ok {
		testAccProvider.With(attrProviderUser, fmt.Sprintf(`"%s"`, v))
	}
	if v, ok := os.LookupEnv("TEST_ACC_LINUX_PROVIDER_PASSWORD"); ok {
		testAccProvider.With(attrProviderPassword, fmt.Sprintf(`"%s"`, v))
	}
	if v, ok := os.LookupEnv("TEST_ACC_LINUX_PROVIDER_PRIVATE_KEY"); ok {
		testAccProvider.With(attrProviderPrivateKey, fmt.Sprintf(`"%s"`, v))
	}
}

func testAccPreCheckConnection(t *testing.T) func() {
	return func() {
		conf := map[string]string{
			attrProviderHost: strings.ReplaceAll(testAccProvider[attrProviderHost], `"`, ``),
			attrProviderPort: testAccProvider[attrProviderPort],
		}
		err := (&linux{connInfo: conf}).init(context.Background())
		var errNet net.Error
		if errors.As(err, &errNet) {
			t.Fatalf("ssh connection should be available: %v", err)
		}
	}
}

func TestMain(m *testing.M) {
	testAccInit()
	resource.TestMain(m)
}
