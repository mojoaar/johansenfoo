ALTER TABLE profile ADD COLUMN bio TEXT NOT NULL DEFAULT '';

UPDATE profile SET bio = 'Enterprise IT leader with 18 years of experience designing and operating complex systems across hybrid environments. I lead a global team, operate cloud infrastructure, and automate everything I can — Terraform, OpenTofu, Python, Go, PowerShell. Passionate about ITSM/ESM done right and making technology invisible so people can do their best work.' WHERE id = 1;
