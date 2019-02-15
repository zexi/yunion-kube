package cloudinit

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/util/seclib2"
)

/*
 * cloudconfig
 * Reference:  https://cloudinit.readthedocs.io/en/latest/topics/examples.html
 *
 */

type TSudoPolicy string

const (
	CLOUD_CONFIG_HEADER = "#cloud-config\n"

	USER_SUDO_NOPASSWD = TSudoPolicy("sudo_nopasswd")
	USER_SUDO          = TSudoPolicy("sudo")
	USER_SUDO_DENY     = TSudoPolicy("sudo_deny")
	USER_SUDO_NONE     = TSudoPolicy("")
)

type SWriteFile struct {
	Path        string
	Permissions string
	Owner       string
	Encoding    string
	Content     string
}

type SUser struct {
	Name              string
	Passwd            string
	LockPassword      string
	SshAuthorizedKeys []string
	Sudo              string
}

type SPhoneHome struct {
	Url string
}

type SCloudConfig struct {
	Users       []SUser
	WriteFiles  []SWriteFile
	Runcmd      []string
	Bootcmd     []string
	Packages    []string
	PhoneHome   *SPhoneHome
	DisableRoot int
	SshPwauth   int
}

func NewWriteFile(path string, content string, perm string, owner string, isBase64 bool) SWriteFile {
	f := SWriteFile{}

	f.Path = path
	f.Permissions = perm
	f.Owner = owner
	if isBase64 {
		f.Encoding = "b64"
		f.Content = base64.StdEncoding.EncodeToString([]byte(content))
	} else {
		f.Content = content
	}

	return f
}

func NewUser(name string) SUser {
	u := SUser{Name: name}
	return u
}

func (u *SUser) SudoPolicy(policy TSudoPolicy) *SUser {
	switch policy {
	case USER_SUDO_NOPASSWD:
		u.Sudo = "ALL=(ALL) NOPASSWD:ALL"
	case USER_SUDO:
		u.Sudo = "ALL=(ALL) ALL"
	case USER_SUDO_DENY:
		u.Sudo = "False"
	default:
		u.Sudo = ""
	}
	return u
}

func (u *SUser) SshKey(key string) *SUser {
	if u.SshAuthorizedKeys == nil {
		u.SshAuthorizedKeys = make([]string, 0)
	}
	u.SshAuthorizedKeys = append(u.SshAuthorizedKeys, key)
	return u
}

func (u *SUser) Password(passwd string) *SUser {
	if len(passwd) > 0 {
		hash, err := seclib2.GeneratePassword(passwd)
		if err != nil {
			log.Errorf("GeneratePassword error %s", err)
		} else {
			u.Passwd = hash
		}
		u.LockPassword = "false"
	}
	return u
}

func (conf *SCloudConfig) UserData() string {
	var buf bytes.Buffer
	jsonConf := jsonutils.Marshal(conf)
	buf.WriteString(CLOUD_CONFIG_HEADER)
	buf.WriteString(jsonConf.YAMLString())
	return buf.String()
}

func (conf *SCloudConfig) UserDataBase64() string {
	data := conf.UserData()
	return base64.StdEncoding.EncodeToString([]byte(data))
}

func ParseUserDataBase64(b64data string) (*SCloudConfig, error) {
	data, err := base64.StdEncoding.DecodeString(b64data)
	if err != nil {
		return nil, err
	}
	return ParseUserData(string(data))
}

func ParseUserData(data string) (*SCloudConfig, error) {
	if !strings.HasPrefix(data, CLOUD_CONFIG_HEADER) {
		msg := "invalid userdata, not starting with #cloud-config"
		log.Errorf(msg)
		return nil, fmt.Errorf(msg)
	}
	jsonConf, err := jsonutils.ParseYAML(data)
	if err != nil {
		log.Errorf("parse userdata yaml error %s", err)
		return nil, err
	}
	config := SCloudConfig{}
	err = jsonConf.Unmarshal(&config)
	if err != nil {
		log.Errorf("unable to unmarchal userdata %s", err)
		return nil, err
	}
	return &config, nil
}

func (conf *SCloudConfig) MergeUser(u SUser) {
	for i := 0; i < len(conf.Users); i += 1 {
		if u.Name == conf.Users[i].Name {
			// find user, merge keys
			for j := 0; j < len(u.SshAuthorizedKeys); j += 1 {
				if !utils.IsInStringArray(u.SshAuthorizedKeys[j], conf.Users[i].SshAuthorizedKeys) {
					conf.Users[i].SshAuthorizedKeys = append(conf.Users[i].SshAuthorizedKeys, u.SshAuthorizedKeys[j])
				}
			}
			return
		}
	}
	// no such user
	conf.Users = append(conf.Users, u)
}

func (conf *SCloudConfig) MergeWriteFile(f SWriteFile, replace bool) {
	for i := 0; i < len(conf.WriteFiles); i += 1 {
		if conf.WriteFiles[i].Path == f.Path {
			// find file
			if replace {
				conf.WriteFiles[i].Content = f.Content
				conf.WriteFiles[i].Encoding = f.Encoding
				conf.WriteFiles[i].Owner = f.Owner
				conf.WriteFiles[i].Permissions = f.Permissions
			}
			return
		}
	}
	// no such file
	conf.WriteFiles = append(conf.WriteFiles, f)
}

func (conf *SCloudConfig) MergeRuncmd(cmd string) {
	if !utils.IsInStringArray(cmd, conf.Runcmd) {
		conf.Runcmd = append(conf.Runcmd, cmd)
	}
}

func (conf *SCloudConfig) MergeBootcmd(cmd string) {
	if !utils.IsInStringArray(cmd, conf.Bootcmd) {
		conf.Bootcmd = append(conf.Bootcmd, cmd)
	}
}

func (conf *SCloudConfig) MergePackage(pkg string) {
	if !utils.IsInStringArray(pkg, conf.Packages) {
		conf.Packages = append(conf.Packages, pkg)
	}
}

func (conf *SCloudConfig) Merge(conf2 *SCloudConfig) {
	for _, u := range conf2.Users {
		conf.MergeUser(u)
	}
	for _, f := range conf2.WriteFiles {
		conf.MergeWriteFile(f, false)
	}
	for _, c := range conf2.Runcmd {
		conf.MergeRuncmd(c)
	}
	for _, c := range conf2.Bootcmd {
		conf.MergeBootcmd(c)
	}
	for _, p := range conf2.Packages {
		conf.MergePackage(p)
	}
}
