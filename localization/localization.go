package localization

import (
	"github.com/gazoon/go-utils/logging"
	"github.com/leonelquinteros/gotext"
	"github.com/pkg/errors"
	"io/ioutil"
	"path"
	"strings"
)

var (
	logger = logging.WithPackage("localization")
)

type Manager struct {
	locales map[string]*gotext.Locale
}

func NewManager(localesPath string) (*Manager, error) {
	locales := map[string]*gotext.Locale{}

	localesDirs, err := ioutil.ReadDir(localesPath)
	if err != nil {
		return nil, errors.Wrap(err, "localization: can't read locales dir")
	}
	for _, dir := range localesDirs {
		if !dir.IsDir() {
			continue
		}
		language := dir.Name()
		domainsPath := path.Join(localesPath, language, "LC_MESSAGES")
		locale := gotext.NewLocale(localesPath, language)
		domainsFiles, err := ioutil.ReadDir(domainsPath)
		if err != nil {
			return nil, errors.Wrap(err, "localization: can't read po files dir")
		}
		for _, file := range domainsFiles {
			fileName := file.Name()
			if file.IsDir() || path.Ext(fileName) != ".po" {
				continue
			}
			domainName := strings.TrimSuffix(fileName, ".po")
			locale.AddDomain(domainName)
		}
		locales[language] = locale
	}
	return &Manager{locales: locales}, nil
}

func (self *Manager) Gettext(lang, msgid string, vars ...interface{}) string {
	return self.GettextD(lang, "default", msgid, vars...)
}

func (self *Manager) GettextD(lang, domain, msgid string, vars ...interface{}) string {
	locale, ok := self.locales[lang]
	if !ok {
		logger.WithField("language", lang).Error("Unknown language")
		return msgid
	}
	return locale.GetD(domain, msgid, vars...)
}
