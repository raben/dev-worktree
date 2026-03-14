package container

const LabelManaged = "dev-cli"

func Labels() map[string]string {
	return map[string]string{
		LabelManaged: "true",
	}
}
