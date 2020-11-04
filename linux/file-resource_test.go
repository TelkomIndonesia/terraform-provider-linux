package linux

import (
	"regexp"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestAccLinuxFileBasic(t *testing.T) {
	conf1 := tfConf{
		Provider: testAccProvider,
		File:     tNewTFMapFile().Without("owner", "group", "mode"),
	}
	conf2 := tfConf{
		Provider: testAccProvider,
		File:     conf1.File.Copy().With("content", `"test"`),
	}
	conf3 := tfConf{
		Provider: testAccProvider,
		File:     tNewTFMapFile(),
		Extra:    tfmap{"path_previous": conf1.File["path"]},
	}

	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{"null": {}},
		PreCheck:          testAccPreCheckConnection(t),
		Providers:         testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxFileBasicConfig(t, conf1),
			},
			{
				Config: testAccLinuxFileBasicConfig(t, conf2),
			},
			{
				Config: testAccLinuxFileBasicConfig(t, conf3),
			},
		},
	})
}

func testAccLinuxFileBasicConfig(t *testing.T, conf tfConf) (s string) {
	tf := heredoc.Doc(`
		provider "linux" {
		    {{- .Provider.Serialize | nindent 4 }}
		}

		resource "null_resource" "destroy_validator" {
		    connection {
		        type = "ssh"
		        {{- .Provider.Serialize | nindent 8 }}
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
		    depends_on = [ null_resource.destroy_validator ]  
		
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
		        {{- .Provider.Serialize | nindent 8 }}
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

func TestAccLinuxFileOverride(t *testing.T) {
	conf1 := tfConf{
		Provider: testAccProvider,
		File:     tNewTFMapFile(),
	}
	conf2 := tfConf{
		Provider: testAccProvider,
		File:     conf1.File.Copy().With("overwrite", "true"),
	}
	conf3 := tfConf{
		Provider: testAccProvider,
		File:     tNewTFMapFile(),
		Extra:    tfmap{"path_previous": conf1.File["path"]},
	}
	conf4 := tfConf{
		Provider: testAccProvider,
		File:     conf3.File.Copy().With("overwrite", "true"),
		Extra:    tfmap{"path_previous": conf1.File["path"]},
	}

	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{"null": {}},
		PreCheck:          testAccPreCheckConnection(t),
		Providers:         testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccLinuxFileOverrideConfig(t, conf1),
				ExpectError: regexp.MustCompile(" exist"),
			},
			{
				Config: testAccLinuxFileOverrideConfig(t, conf2),
			},
			{
				Config:      testAccLinuxFileOverrideConfig(t, conf3),
				ExpectError: regexp.MustCompile(" exist"),
			},
			{
				Config: testAccLinuxFileOverrideConfig(t, conf4),
			},
		},
	})
}

func testAccLinuxFileOverrideConfig(t *testing.T, conf tfConf) (s string) {
	tf := heredoc.Doc(`
		provider "linux" {
		    {{- .Provider.Serialize | nindent 4 }}
		}

		resource "null_resource" "existing_file" {
		    triggers = {
		        path = {{ .File.path }}
		    }
		    connection {
		        type = "ssh"
		        {{- .Provider.Serialize | nindent 8 }}
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
		    depends_on = [ null_resource.existing_file ]  
		
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
		        {{- .Provider.Serialize | nindent 8 }}
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

func TestAccLinuxFileIgnoreContent(t *testing.T) {
	conf1 := tfConf{
		Provider: testAccProvider,
		File:     tNewTFMapFile().With("ignore_content", "true"),
	}
	conf2 := tfConf{
		Provider: testAccProvider,
		File:     tNewTFMapFile().With("ignore_content", "true"),
		Extra:    tfmap{"path_previous": conf1.File["path"]},
	}
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{"null": {}},
		PreCheck:          testAccPreCheckConnection(t),
		Providers:         testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxFileIgnoreContentConfig(t, conf1),
			},
			{
				Config: testAccLinuxFileIgnoreContentConfig(t, conf2),
			},
		},
	})
}

func testAccLinuxFileIgnoreContentConfig(t *testing.T, conf tfConf) (s string) {
	tf := heredoc.Doc(`
		provider "linux" {
		    {{- .Provider.Serialize | nindent 4 }}
		}

		locals {
		    new_content = "new content"
		}

		resource "linux_file" "file" {
		    {{- .File.Serialize | nindent 4 }}

		    connection {
		        type = "ssh"
		        {{- .Provider.Serialize | nindent 8 }}
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
		        {{- .Provider.Serialize | nindent 8 }}
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

func TestAccLinuxFileRecyclePath(t *testing.T) {
	conf1 := tfConf{
		Provider: testAccProvider,
		File:     tNewTFMapFile().With("recycle_path", `"/tmp/recycle"`),
	}
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{"null": {}},
		PreCheck:          testAccPreCheckConnection(t),
		Providers:         testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxFileRecyclePathConfig(t, conf1),
			},
		},
	})
}

func testAccLinuxFileRecyclePathConfig(t *testing.T, conf tfConf) (s string) {
	tf := heredoc.Doc(`
		provider "linux" {
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
		        {{- .Provider.Serialize | nindent 8 }}
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
		    depends_on = [ null_resource.destroy_checker ]
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
		        {{- .Provider.Serialize | nindent 8 }}
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
