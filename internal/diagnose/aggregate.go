package diagnose

func AggregateSeverity(issues []Issue) Severity {
	sev := SeverityOK

	for _, i := range issues {
		switch i.Severity {
		case SeverityFatal:
			return SeverityFatal
		case SeverityError:
			sev = SeverityError
		case SeverityWarn:
			if sev == SeverityOK {
				sev = SeverityWarn
			}
		}
	}

	return sev
}
