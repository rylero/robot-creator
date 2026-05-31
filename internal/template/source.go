package template

var validTypes = []string{
	"flywheel", "pivot", "roller", "arm", "elevator", "turret", "generic", "manipulator",
}

// TemplateSource abstracts where templates come from.
// EmbeddedTemplateSource is the MVP impl; LocalTemplateSource is planned post-MVP.
type TemplateSource interface {
	GetTemplate(subsystemType, fileName string) ([]byte, error)
	ListTypes() []string
}

func IsValidType(t string) bool {
	for _, v := range validTypes {
		if v == t {
			return true
		}
	}
	return false
}
