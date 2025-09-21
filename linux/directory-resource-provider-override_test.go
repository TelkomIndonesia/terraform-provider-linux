package linux

import (
	"regexp"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestAccLinuxDirectoryProviderOverrideBasic(t *testing.T) {
	conf1 := tfConf{
		Provider:         testAccOverridenProvider,
		ProviderOverride: testAccProvider.Copy().With("id", `"someid"`),
		Directory:        tNewTFMapDirectory().Without(attrDirectoryOwner, attrDirectoryGroup, attrDirectoryMode),
	}
	conf2 := conf1.Copy(func(tc *tfConf) {
		tc.Directory.With(attrDirectoryMode, "700")
	})
	conf3 := tfConf{
		Provider:         testAccOverridenProvider,
		ProviderOverride: testAccProvider.Copy().With("id", `"someid"`),
		Directory:        tNewTFMapDirectory(),
		Extra:            tfmap{"path_previous": conf1.Directory[attrDirectoryPath]},
	}

	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{"null": {}},
		Providers:         testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxDirectoryProviderOverrideBasicConfig(t, conf1),
			},
			{
				Config: testAccLinuxDirectoryProviderOverrideBasicConfig(t, conf2),
			},
			{
				Config: testAccLinuxDirectoryProviderOverrideBasicConfig(t, conf3),
			},
		},
	})
}

func testAccLinuxDirectoryProviderOverrideBasicConfig(t *testing.T, conf tfConf) (s string) {
	tf := heredoc.Doc(`
		provider "linux" {
			alias = "test"
		    {{- .Provider.Serialize | nindent 4 }}
		}

		resource "null_resource" "destroy_validator" {
		    connection {
		        type = "ssh"
		        {{- ((.ProviderOverride.Copy).Without "id").Serialize | nindent 8 }}
		    }
		    provisioner "remote-exec" {
		        inline = [
		            <<-EOF
		                rm -rf {{ .Directory.path }} || true
		            EOF
		        ]
		    }
		    provisioner "remote-exec" {
		        when = destroy
		        inline = [
		            <<-EOF
		                [ ! -e {{ .Directory.path }} ] || exit 100
		            EOF
		        ]
		    }
		}

		resource "linux_directory" "directory" {
			provider = linux.test
		    depends_on = [ null_resource.destroy_validator ]  
		
		    provider_override {
		        {{- .ProviderOverride.Serialize | nindent 8 }}
		    }
		    {{- .Directory.Serialize | nindent 4 }}
		}

		resource "null_resource" "file" {
		    triggers = {
		        name = "exist"
		        content = "i need to exist"
		    }
		    
		    connection {
		        type = "ssh"
		        {{- ((.ProviderOverride.Copy).Without "id").Serialize | nindent 8 }}
		    }
		    provisioner "file" {
		        content = self.triggers["content"]
		        destination = "${linux_directory.directory.path}/${self.triggers["name"]}"
		    }
		}

		resource "null_resource" "create_validator" {
		    triggers = {
		        {{- range $key, $value := .Directory }}
		            {{- $key | nindent 8 }} = linux_directory.directory.{{ $key }}
		        {{- end}}
		
		        path_previous = {{ .Extra.path_previous | default "0"}}
		    }
		    connection {
		        type = "ssh"
		        {{- ((.ProviderOverride.Copy).Without "id").Serialize | nindent 8 }}
		    }
		    provisioner "remote-exec" {
		        inline = [
		            <<-EOF
		                [ ! -e '${self.triggers["path_previous"]}' ] || exit 101
		                [ -d '${self.triggers["path"]}' ] || exit 102
		                [ "$(cat '${self.triggers["path"]}/${null_resource.file.triggers["name"]}')" == "${null_resource.file.triggers["content"]}" ] || exit 103
		
		                [ "$( stat -c %u '${self.triggers["path"]}' )" == "{{ .Directory.owner | default 0 }}" ] || exit 104
		                [ "$( stat -c %g '${self.triggers["path"]}' )" == "{{ .Directory.group | default 0 }}" ] || exit 105
		                [ "$( stat -c %a '${self.triggers["path"]}' )" == {{ .Directory.mode | default "755" }} ] || exit 106
		            EOF
		        ]
		    }
		}
	`)
	s, err := conf.compile(tf)
	t.Log(s)
	require.NoError(t, err, "compile template failed")
	return
}

func TestAccLinuxDirectoryProviderOverrideOverwrite(t *testing.T) {
	conf1 := tfConf{
		Provider:         testAccOverridenProvider,
		ProviderOverride: testAccProvider.Copy().With("id", `"someid"`),
		Directory:        tNewTFMapDirectory(),
	}
	conf2 := conf1.Copy(func(tc *tfConf) {
		tc.Directory.With(attrDirectoryOverwrite, "true")
	})
	conf3 := tfConf{
		Provider:         testAccOverridenProvider,
		ProviderOverride: testAccProvider.Copy().With("id", `"someid"`),
		Directory:        tNewTFMapDirectory(),
		Extra:            tfmap{"path_previous": conf1.Directory[attrDirectoryPath]},
	}
	conf4 := conf3.Copy(func(tc *tfConf) {
		tc.Directory.With(attrDirectoryOverwrite, "true")
	})
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{"null": {}},
		PreCheck:          testAccPreCheckConnection(t),
		Providers:         testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccLinuxDirectoryProviderOverrideeOverwriteConfig(t, conf1),
				ExpectError: regexp.MustCompile(" exist"),
			},
			{
				Config: testAccLinuxDirectoryProviderOverrideeOverwriteConfig(t, conf2),
			},
			{
				Config:      testAccLinuxDirectoryProviderOverrideeOverwriteConfig(t, conf3),
				ExpectError: regexp.MustCompile(" exist"),
			},
			{
				Config: testAccLinuxDirectoryProviderOverrideeOverwriteConfig(t, conf4),
			},
		},
	})
}

func testAccLinuxDirectoryProviderOverrideeOverwriteConfig(t *testing.T, conf tfConf) (s string) {
	tf := heredoc.Doc(`
		provider "linux" {
			alias = "test"
		    {{- .Provider.Serialize | nindent 4 }}
		}

		resource "null_resource" "existing" {
		    connection {
		        type = "ssh"
		        {{- ((.ProviderOverride.Copy).Without "id").Serialize | nindent 8 }}
		    }
		    triggers = {
		        path = {{ .Directory.path }}
		    }
		    provisioner "remote-exec" {
		        inline = [
		            <<-EOF
		                mkdir -p "${self.triggers["path"]}"
		                echo -n "existing" > "${self.triggers["path"]}/existing"
		            EOF
		        ]
		    }
		}

		resource "linux_directory" "directory" {
			provider = linux.test
		    depends_on = [ null_resource.existing ]  
		
		    provider_override {
		        {{- .ProviderOverride.Serialize | nindent 8 }}
		    }
		    {{- .Directory.Serialize | nindent 4 }}
		}

		resource "null_resource" "create_validator" {
		    triggers = {
		        {{- range $key, $value := .Directory }}
		            {{- $key | nindent 8 }} = linux_directory.directory.{{ $key }}
		        {{- end}}
		
		        path_previous = {{ .Extra.path_previous | default "0"}}
		    }
		    connection {
		        type = "ssh"
		        {{- ((.ProviderOverride.Copy).Without "id").Serialize | nindent 8 }}
		    }
		    provisioner "remote-exec" {
		        inline = [
		            <<-EOF
		                [ ! -e '${self.triggers["path_previous"]}' ] || exit 101
		
		                [ -d '${self.triggers["path"]}' ] || exit 102
		                [ "$(cat '${self.triggers["path"]}/existing')" == "existing" ] || exit 103
		
		                [ "$( stat -c %u '${self.triggers["path"]}' )" == "{{ .Directory.owner | default 0 }}" ] || exit 104
		                [ "$( stat -c %g '${self.triggers["path"]}' )" == "{{ .Directory.group | default 0 }}" ] || exit 105
		                [ "$( stat -c %a '${self.triggers["path"]}' )" == {{ .Directory.mode | default "755" }} ] || exit 106
		            EOF
		        ]
		    }
		}
	`)
	s, err := conf.compile(tf)
	t.Log(s)
	require.NoError(t, err, "compile template failed")
	return
}

func TestAccLinuxDirectoryProviderOverrideRecyclePath(t *testing.T) {
	conf1 := tfConf{
		Provider:         testAccOverridenProvider,
		ProviderOverride: testAccProvider.Copy().With("id", `"someid"`),
		Directory:        tNewTFMapDirectory().With(attrDirectoryRecyclePath, `"/tmp/recycle"`),
	}
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{"null": {}},
		PreCheck:          testAccPreCheckConnection(t),
		Providers:         testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxDirectoryProviderOverrideRecyclePathConfig(t, conf1),
			},
		},
	})
}

func testAccLinuxDirectoryProviderOverrideRecyclePathConfig(t *testing.T, conf tfConf) (s string) {
	tf := heredoc.Doc(`
		provider "linux" {
			alias = "test"
		    {{- .Provider.Serialize | nindent 4 }}
		}

		locals {
		    filename = "exist"
		}

		resource "null_resource" "destroy_validator" {
		    connection {
		        type = "ssh"
		        {{- ((.ProviderOverride.Copy).Without "id").Serialize | nindent 8 }}
		    }
		    triggers = {
			    filename = local.filename
		        recycle_path = {{ .Directory.recycle_path }}
		    }
		    provisioner "remote-exec" {
		        when = destroy
		        inline = [
		            <<-EOF
		                [ ! -e {{ .Directory.path }} ] || exit 100
		                find "${self.triggers["recycle_path"]}" -name "${ basename( {{ .Directory.path}} ) }" | grep . || exit 101
		                find "${self.triggers["recycle_path"]}" -name "${ self.triggers["filename"] }" |  grep "${ basename( {{ .Directory.path}} ) }/${ self.triggers["filename"] }" | grep . || exit 102
		                rm -rf "${self.triggers["recycle_path"]}" || exit 103
		            EOF
		        ]
		    }
		}

		resource "linux_directory" "directory" {
			provider = linux.test
		    depends_on = [ null_resource.destroy_validator ]  
		
		    provider_override {
		        {{- .ProviderOverride.Serialize | nindent 8 }}
		    }
		    {{- .Directory.Serialize | nindent 4 }}
		}

		resource "null_resource" "file" {
		    triggers = {
		        name = local.filename
		        content = "i need to exist"
		    }
		    
		    connection {
		        type = "ssh"
		        {{- ((.ProviderOverride.Copy).Without "id").Serialize | nindent 8 }}
		    }
		    provisioner "file" {
		        content = self.triggers["content"]
		        destination = "${linux_directory.directory.path}/${self.triggers["name"]}"
		    }
		}

		resource "null_resource" "create_validator" {
		    triggers = {
		        {{- range $key, $value := .Directory }}
		            {{- $key | nindent 8 }} = linux_directory.directory.{{ $key }}
		        {{- end}}
		
		        path_previous = {{ .Extra.path_previous | default "0"}}
		    }
		    connection {
		        type = "ssh"
		        {{- ((.ProviderOverride.Copy).Without "id").Serialize | nindent 8 }}
		    }
		    provisioner "remote-exec" {
		        inline = [
		            <<-EOF
		                [ ! -e '${self.triggers["path_previous"]}' ] || exit 104
		                
		                [ -d '${self.triggers["path"]}' ] || exit 105
		                [ "$(cat '${self.triggers["path"]}/${null_resource.file.triggers["name"]}')" == "${null_resource.file.triggers["content"]}" ] || exit 106
		
		                [ "$( stat -c %u '${self.triggers["path"]}' )" == "{{ .Directory.owner | default 0 }}" ] || exit 107
		                [ "$( stat -c %g '${self.triggers["path"]}' )" == "{{ .Directory.group | default 0 }}" ] || exit 108
		                [ "$( stat -c %a '${self.triggers["path"]}' )" == {{ .Directory.mode | default "755" }} ] || exit 109
		            EOF
		        ]
		    }
		}
	`)
	s, err := conf.compile(tf)
	t.Log(s)
	require.NoError(t, err, "compile template failed")
	return
}
