package linux

import (
	"regexp"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestAccLinuxDirectoryBasic(t *testing.T) {

	conf1 := tfConf{
		Provider:  provider,
		Directory: tNewTFMapDirectory().Without("owner", "group", "mode"),
	}
	conf2 := tfConf{
		Provider:  provider,
		Directory: tNewTFMapDirectory(),
		Extra:     tfmap{"path_previous": conf1.Directory["path"]},
	}

	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"null": {},
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxDirectoryBasicConfig(t, conf1),
			},
			{
				Config: testAccLinuxDirectoryBasicConfig(t, conf2),
			},
		},
	})
}

func testAccLinuxDirectoryBasicConfig(t *testing.T, conf tfConf) (s string) {
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
		    depends_on = [ null_resource.destroy_validator ]  
		
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
		        {{- .Provider.Serialize | nindent 8 }}
		    }
		    provisioner "remote-exec" {
		        inline = [
		            <<-EOF
		                [ ! -e '${self.triggers.path_previous}' ] || exit 101
		                [ -d '${self.triggers.path}' ] || exit 102
		
		                [ "$( stat -c %u '${self.triggers.path}' )" == "{{ .Directory.owner | default 0 }}" ] || exit 103
		                [ "$( stat -c %g '${self.triggers.path}' )" == "{{ .Directory.group | default 0 }}" ] || exit 104
		                [ "$( stat -c %a '${self.triggers.path}' )" == {{ .Directory.mode | default "755" }} ] || exit 105
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

func TestAccLinuxDirectoryOverride(t *testing.T) {
	conf1 := tfConf{
		Provider:  provider,
		Directory: tNewTFMapDirectory(),
	}
	conf2 := tfConf{
		Provider:  provider,
		Directory: conf1.Directory.Copy().With("overwrite", "true"),
	}
	conf3 := tfConf{
		Provider:  provider,
		Directory: tNewTFMapDirectory(),
		Extra:     tfmap{"path_previous": conf1.Directory["path"]},
	}
	conf4 := tfConf{
		Provider:  provider,
		Directory: conf3.Directory.Copy().With("overwrite", "true"),
		Extra:     tfmap{"path_previous": conf1.Directory["path"]},
	}
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"null": {},
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccLinuxDirectoryeOverrideConfig(t, conf1),
				ExpectError: regexp.MustCompile(" exist"),
			},
			{
				Config: testAccLinuxDirectoryeOverrideConfig(t, conf2),
			},
			{
				Config:      testAccLinuxDirectoryeOverrideConfig(t, conf3),
				ExpectError: regexp.MustCompile(" exist"),
			},
			{
				Config: testAccLinuxDirectoryeOverrideConfig(t, conf4),
			},
		},
	})
}

func testAccLinuxDirectoryeOverrideConfig(t *testing.T, conf tfConf) (s string) {
	tf := heredoc.Doc(`
		provider "linux" {
		    {{- .Provider.Serialize | nindent 4 }}
		}

		resource "null_resource" "existing" {
		    connection {
		        type = "ssh"
		        {{- .Provider.Serialize | nindent 8 }}
			}
			triggers = {
				path = {{ .Directory.path }}
			}
			provisioner "remote-exec" {
				inline = [
					<<-EOF
						mkdir -p "${self.triggers.path}"
					EOF
				]
			}
		}

		resource "linux_directory" "directory" {
		    depends_on = [ null_resource.existing ]  
		
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
		        {{- .Provider.Serialize | nindent 8 }}
		    }
		    provisioner "remote-exec" {
		        inline = [
		            <<-EOF
		                [ ! -e '${self.triggers.path_previous}' ] || exit 101
		                [ -d '${self.triggers.path}' ] || exit 102
		
		                [ "$( stat -c %u '${self.triggers.path}' )" == "{{ .Directory.owner | default 0 }}" ] || exit 103
		                [ "$( stat -c %g '${self.triggers.path}' )" == "{{ .Directory.group | default 0 }}" ] || exit 104
		                [ "$( stat -c %a '${self.triggers.path}' )" == {{ .Directory.mode | default "755" }} ] || exit 105
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

func TestAccLinuxDirectoryRecyclePath(t *testing.T) {
	conf1 := tfConf{
		Provider:  provider,
		Directory: tNewTFMapDirectory().With("recycle_path", `"/tmp/recycle"`),
	}
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"null": {},
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxDirectoryRecyclePathConfig(t, conf1),
			},
		},
	})
}

func testAccLinuxDirectoryRecyclePathConfig(t *testing.T, conf tfConf) (s string) {
	tf := heredoc.Doc(`
		provider "linux" {
		    {{- .Provider.Serialize | nindent 4 }}
		}

		resource "null_resource" "destroy_validator" {
		    connection {
		        type = "ssh"
		        {{- .Provider.Serialize | nindent 8 }}
		    }
		    triggers = {
		        recycle_path = {{ .Directory.recycle_path }}
		    }
		    provisioner "remote-exec" {
				when = destroy
		        inline = [
		            <<-EOF
		                [ ! -e {{ .Directory.path }} ] || exit 100
		                find "${self.triggers.recycle_path}" -name  "${ basename( {{ .Directory.path}} ) }" | grep . || exit 101
		                rm -rf "${self.triggers.recycle_path}" || exit 102
		            EOF
		        ]
		    }
		}

		resource "linux_directory" "directory" {
		    depends_on = [ null_resource.destroy_validator ]  
		
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
		        {{- .Provider.Serialize | nindent 8 }}
		    }
		    provisioner "remote-exec" {
		        inline = [
		            <<-EOF
		                [ ! -e '${self.triggers.path_previous}' ] || exit 103
		                [ -d '${self.triggers.path}' ] || exit 104
		
		                [ "$( stat -c %u '${self.triggers.path}' )" == "{{ .Directory.owner | default 0 }}" ] || exit 105
		                [ "$( stat -c %g '${self.triggers.path}' )" == "{{ .Directory.group | default 0 }}" ] || exit 106
		                [ "$( stat -c %a '${self.triggers.path}' )" == {{ .Directory.mode | default "755" }} ] || exit 107
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
