package linux

import (
	"regexp"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestAccLinuxFileProviderOverrideBasic(t *testing.T) {
	conf1 := tfConf{
		Provider:         testAccOverridenProvider,
		ProviderOverride: testAccProvider.Copy().With("id", `"someid"`),
		File:             tNewTFMapFile().Without(attrFileOwner, attrFileGroup, attrFileMode),
	}
	conf2 := conf1.Copy(func(tc *tfConf) {
		tc.File.With(attrFileContent, `"test"`)
	})
	conf3 := tfConf{
		Provider:         testAccOverridenProvider,
		ProviderOverride: testAccProvider.Copy().With("id", `"someid"`),
		File:             tNewTFMapFile(),
		Extra:            tfmap{"path_previous": conf1.File[attrFilePath]},
	}

	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{"null": {}},
		PreCheck:          testAccPreCheckConnection(t),
		Providers:         testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxFileProviderOverrideBasicConfig(t, conf1),
			},
			{
				Config: testAccLinuxFileProviderOverrideBasicConfig(t, conf2),
			},
			{
				Config: testAccLinuxFileProviderOverrideBasicConfig(t, conf3),
			},
		},
	})
}

func testAccLinuxFileProviderOverrideBasicConfig(t *testing.T, conf tfConf) (s string) {
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
		        when = destroy
		        inline = [
		            <<-EOF
		            [ ! -e {{ .File.path }} ] || exit 100
		            EOF
		        ]
		    }
		}

		resource "linux_file" "file" {
			provider = "linux.test"
		    depends_on = [ null_resource.destroy_validator ]  
			
		    provider_override {
		        {{- .ProviderOverride.Serialize | nindent 8 }}
		    }
			
		    {{- .File.Serialize | nindent 4 }}
		}

		resource "null_resource" "create_validator" {
		    triggers = {
		        {{- range $key, $value := .File }}
		            {{- $key | nindent 8 }} = linux_file.file.{{ $key }}
		        {{- end}}
		
		        path_previous = {{ .Extra.path_previous | default "0"}}
		        path_compare = "${linux_file.file.path}.compare"
		    }
		    connection {
		        type = "ssh"
		        {{- ((.ProviderOverride.Copy).Without "id").Serialize | nindent 8 }}
		    }
		    provisioner "file" {
		        content = self.triggers["content"]
		        destination = self.triggers["path_compare"]
		    }
		    provisioner "remote-exec" {
		        inline = [
		            <<-EOF
		                [ ! -e "${self.triggers["path_previous"]}"  ] || exit 101
		
		                cmp -s "${self.triggers["path"]}" "${self.triggers["path_compare"]}" || exit 102
		                [ "$( stat -c %u '${self.triggers["path"]}' )" == "{{ .File.owner | default 0 }}" ] || exit 103
		                [ "$( stat -c %g '${self.triggers["path"]}' )" == "{{ .File.group | default 0 }}" ] || exit 104
		                [ "$( stat -c %a '${self.triggers["path"]}' )" == {{ .File.mode | default 644 }} ] || exit 105
		            EOF
		        ]
		    }
		    provisioner "remote-exec" {
		        when = destroy
		        inline = [ "rm -f '${self.triggers["path_compare"]}'" ]
		    }
		}
	`)

	s, err := conf.compile(tf)
	t.Log(s)
	require.NoError(t, err, "compile template failed")
	return
}

func TestAccLinuxFileProviderOverrideOverride(t *testing.T) {
	conf1 := tfConf{
		Provider:         testAccOverridenProvider,
		ProviderOverride: testAccProvider.Copy().With("id", `"someid"`),
		File:             tNewTFMapFile(),
	}
	conf2 := conf1.Copy(func(tc *tfConf) {
		tc.File.With(attrFileOverwrite, "true")
	})
	conf3 := tfConf{
		Provider:         testAccOverridenProvider,
		ProviderOverride: testAccProvider.Copy().With("id", `"someid"`),
		File:             tNewTFMapFile(),
		Extra:            tfmap{"path_previous": conf1.File[attrFilePath]},
	}
	conf4 := conf3.Copy(func(tc *tfConf) {
		tc.File.With(attrFileOverwrite, "true")
	})

	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{"null": {}},
		PreCheck:          testAccPreCheckConnection(t),
		Providers:         testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccLinuxFileProviderOverrideOverrideConfig(t, conf1),
				ExpectError: regexp.MustCompile(" exist"),
			},
			{
				Config: testAccLinuxFileProviderOverrideOverrideConfig(t, conf2),
			},
			{
				Config:      testAccLinuxFileProviderOverrideOverrideConfig(t, conf3),
				ExpectError: regexp.MustCompile(" exist"),
			},
			{
				Config: testAccLinuxFileProviderOverrideOverrideConfig(t, conf4),
			},
		},
	})
}

func testAccLinuxFileProviderOverrideOverrideConfig(t *testing.T, conf tfConf) (s string) {
	tf := heredoc.Doc(`
		provider "linux" {
			alias = "test"
		    {{- .Provider.Serialize | nindent 4 }}
		}

		resource "null_resource" "existing_file" {
		    triggers = {
		        path = {{ .File.path }}
		    }
		    connection {
		        type = "ssh"
		        {{- ((.ProviderOverride.Copy).Without "id").Serialize | nindent 8 }}
		    }
		    provisioner "remote-exec" {
		        inline = [ "mkdir -p ${ dirname(self.triggers["path"]) }" ]
		    }
		    provisioner "file" {
		        content = "existing"
		        destination = self.triggers["path"]
		    }
		}

		resource "linux_file" "file" {
			provider = "linux.test"
		    depends_on = [ null_resource.existing_file ]  
		
		    provider_override {
		        {{- .ProviderOverride.Serialize | nindent 8 }}
		    }
		    {{- .File.Serialize | nindent 4 }}
		}

		resource "null_resource" "create_validator" {
		    triggers = {
		        {{- range $key, $value := .File }}
		            {{- $key | nindent 8 }} = linux_file.file.{{ $key }}
		        {{- end}}
		
		        path_previous = {{ .Extra.path_previous | default "0"}}
		        path_compare = "${linux_file.file.path}.compare"
		    }
		    connection {
		        type = "ssh"
		        {{- ((.ProviderOverride.Copy).Without "id").Serialize | nindent 8 }}
		    }
		    provisioner "file" {
		        content = self.triggers["content"]
		        destination = self.triggers["path_compare"]
		    }
		    provisioner "remote-exec" {
		        inline = [
		            <<-EOF
		                [ ! -e "${self.triggers["path_previous"]}"  ] || exit 101
		
		                cmp -s "${self.triggers["path"]}" "${self.triggers["path_compare"]}" || exit 102
		                [ "$( stat -c %u '${self.triggers["path"]}' )" == "{{ .File.owner | default 0 }}" ] || exit 103
		                [ "$( stat -c %g '${self.triggers["path"]}' )" == "{{ .File.group | default 0 }}" ] || exit 104
		                [ "$( stat -c %a '${self.triggers["path"]}' )" == {{ .File.mode | default 644 }} ] || exit 105
		            EOF
		        ]
		    }
		    provisioner "remote-exec" {
		        when = destroy
		        inline = [ "rm -f '${self.triggers["path_compare"]}'" ]
		    }
		}
	`)

	s, err := conf.compile(tf)
	t.Log(s)
	require.NoError(t, err, "compile template failed")
	return
}

func TestAccLinuxFileProviderOverrideIgnoreContent(t *testing.T) {
	conf1 := tfConf{
		Provider:         testAccOverridenProvider,
		ProviderOverride: testAccProvider.Copy().With("id", `"someid"`),
		File:             tNewTFMapFile().With("ignore_content", "true"),
	}
	conf2 := tfConf{
		Provider:         testAccOverridenProvider,
		ProviderOverride: testAccProvider.Copy().With("id", `"someid"`),
		File:             tNewTFMapFile().With("ignore_content", "true"),
		Extra:            tfmap{"path_previous": conf1.File["path"]},
	}
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{"null": {}},
		PreCheck:          testAccPreCheckConnection(t),
		Providers:         testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxFileProviderOverrideIgnoreContentConfig(t, conf1),
			},
			{
				Config: testAccLinuxFileProviderOverrideIgnoreContentConfig(t, conf2),
			},
		},
	})
}

func testAccLinuxFileProviderOverrideIgnoreContentConfig(t *testing.T, conf tfConf) (s string) {
	tf := heredoc.Doc(`
		provider "linux" {
			alias = "test"
		    {{- .Provider.Serialize | nindent 4 }}
		}

		locals {
		    new_content = "new content"
		}

		resource "linux_file" "file" {
			provider = "linux.test"
		    provider_override {
		        {{- .ProviderOverride.Serialize | nindent 8 }}
		    }
		    {{- .File.Serialize | nindent 4 }}

		    connection {
		        type = "ssh"
		        {{- ((.ProviderOverride.Copy).Without "id").Serialize | nindent 8 }}
		    }
		    
		    provisioner "remote-exec" {
		        inline = [ "echo -n '${local.new_content}' > ${self.path}" ]
		    }
		}

		resource "null_resource" "create_validator" {
		    triggers = {
		        {{- range $key, $value := .File }}
		            {{- $key | nindent 8 }} = linux_file.file.{{ $key }}
		        {{- end}}
		
		        path_previous = {{ .Extra.path_previous | default "0"}}
		        path_compare = "${linux_file.file.path}.compare"
		    }
		    connection {
		        type = "ssh"
		        {{- ((.ProviderOverride.Copy).Without "id").Serialize | nindent 8 }}
		    }
		    provisioner "file" {
		        content = local.new_content
		        destination = self.triggers["path_compare"]
		    }
		    provisioner "remote-exec" {
		        inline = [
		            <<-EOF
		                [ ! -e "${self.triggers["path_previous"]}"  ] || exit 101
		
		                cmp -s "${self.triggers["path"]}" "${self.triggers["path_compare"]}" || exit 102
		                [ "$( stat -c %u '${self.triggers["path"]}' )" == "{{ .File.owner | default 0 }}" ] || exit 103
		                [ "$( stat -c %g '${self.triggers["path"]}' )" == "{{ .File.group | default 0 }}" ] || exit 104
		                [ "$( stat -c %a '${self.triggers["path"]}' )" == {{ .File.mode | default 644 }} ] || exit 105
		            EOF
		        ]
		    }
		    provisioner "remote-exec" {
		        when = destroy
		        inline = [ "rm -f '${self.triggers["path_compare"]}'" ]
		    }
		}
	`)
	s, err := conf.compile(tf)
	t.Log(s)
	require.NoError(t, err, "compile template failed")
	return
}

func TestAccLinuxFileProviderOverrideRecyclePath(t *testing.T) {
	conf1 := tfConf{
		Provider:         testAccOverridenProvider,
		ProviderOverride: testAccProvider.Copy().With("id", `"someid"`),
		File:             tNewTFMapFile().With(attrFileRecyclePath, `"/tmp/recycle"`),
	}
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{"null": {}},
		PreCheck:          testAccPreCheckConnection(t),
		Providers:         testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxFileProviderOverrideRecyclePathConfig(t, conf1),
			},
		},
	})
}

func testAccLinuxFileProviderOverrideRecyclePathConfig(t *testing.T, conf tfConf) (s string) {
	tf := heredoc.Doc(`
		provider "linux" {
			alias = "test"
		    {{- .Provider.Serialize | nindent 4 }}
		}

		locals {
		    recycle_path = {{ .File.recycle_path }}
		}

		resource "null_resource" "destroy_checker" {
		    triggers = {
		        recycle_path = local.recycle_path
		    }
		    connection {
		        type = "ssh"
		        {{- ((.ProviderOverride.Copy).Without "id").Serialize | nindent 8 }}
		    }
		    provisioner "remote-exec" {
		        when = destroy
		        inline = [
		            <<-EOF
		                [ ! -e {{ .File.path }} ] || exit 100
		                find ${self.triggers["recycle_path"]} -name "$(basename "{{ .File.path }}")" | grep . || exit 101
		                rm -rf ${self.triggers["recycle_path"]} || exit 102
		            EOF
		        ]
		    }
		}

		resource "linux_file" "file" {
			provider = "linux.test"
		    depends_on = [ null_resource.destroy_checker ]
		    provider_override {
		        {{- .ProviderOverride.Serialize | nindent 8 }}
		    }
		    {{- .File.Serialize | nindent 4 }}
		}

		resource "null_resource" "create_validator" {
		    triggers = {
		        {{- range $key, $value := .File }}
		            {{- $key | nindent 8 }} = linux_file.file.{{ $key }}
		        {{- end}}
		
		        path_previous = {{ .Extra.path_previous | default "0"}}
		        path_compare = "${linux_file.file.path}.compare"
		    }
		    connection {
		        type = "ssh"
		        {{- ((.ProviderOverride.Copy).Without "id").Serialize | nindent 8 }}
		    }
		    provisioner "file" {
		        content = self.triggers["content"]
		        destination = self.triggers["path_compare"]
		    }
		    provisioner "remote-exec" {
		        inline = [
		            <<-EOF
		                [ ! -e "${self.triggers["path_previous"]}"  ] || exit 101
		
		                cmp -s "${self.triggers["path"]}" "${self.triggers["path_compare"]}" || exit 103
		                [ "$( stat -c %u '${self.triggers["path"]}' )" == "{{ .File.owner | default 0 }}" ] || exit 104
		                [ "$( stat -c %g '${self.triggers["path"]}' )" == "{{ .File.group | default 0 }}" ] || exit 105
		                [ "$( stat -c %a '${self.triggers["path"]}' )" == {{ .File.mode | default 644 }} ] || exit 106
		            EOF
		        ]
		    }
		    provisioner "remote-exec" {
		        when = destroy
		        inline = [ "rm -f '${self.triggers["path_compare"]}'" ]
		    }
		}
	`)
	s, err := conf.compile(tf)
	t.Log(s)
	require.NoError(t, err, "compile template failed")
	return
}
