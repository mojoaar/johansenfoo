INSERT INTO profile (id, name, handle, location, dob, tagline, hero_bio, about_para_1, about_para_2, avatar, updated_at)
VALUES (
    1,
    'Morten Johansen',
    'mojoaar',
    'Denmark',
    '1980-08-13',
    '// enterprise it leader & open-source tinkerer',
    'Building and running complex infrastructure & cloud environments for 18+ years. Spent the last 5 leading global ops @ [JYSK](https://jysk.com) - 15 people, real budgets, and the quiet satisfaction of systems that heal themselves. Automation, architecture, and a few hard-earned lessons hold it all together. Father, husband, open-source tinkerer by night, with the occasional side project that scratches an itch.',
    'Two decades in IT — from Lance Corporal to global ops leader. I''ve sat on both sides of every ticket, outage, and migration. What I''ve learned: good infrastructure is boring by design, automation is cheaper than burnout, and the best IT leaders still remember what it felt like to be on call at 3 AM.',
    'Process that serves people, not the other way around. Clarity beats chaos every time.',
    'avatar.png',
    strftime('%Y-%m-%dT%H:%M:%SZ','now')
);

INSERT INTO social_link (platform, url, label, sort) VALUES
    ('bluesky',  'https://bsky.app/profile/johansen.foo', 'Bluesky',  0),
    ('linkedin', 'https://linkedin.com/in/mojoaar',       'LinkedIn', 1),
    ('mastodon', 'https://floss.social/@mojoaar',         'Mastodon', 2),
    ('github',   'https://github.com/mojoaar',            'GitHub',   3);

INSERT INTO project (name, url, description, icon, is_link, url_label, sort) VALUES
    ('atlascmdb', 'https://github.com/mojoaar/atlascmdb',
     'Open-source CMDB built on Next.js - entity management, relationship graphing, TOTP MFA, SSO/SCIM, rack layouts, and bulk import.',
     'database', 1, 'github.com/mojoaar/atlascmdb', 0),
    ('clutch', 'https://github.com/mojoaar/clutch',
     'Cross-platform desktop AI chat - multi-provider streaming, markdown, file attachments, web fetching, 8 themes, and i18n. Tauri v2 + SvelteKit.',
     'message-square', 1, 'github.com/mojoaar/clutch', 1),
    ('echo', 'https://echo.johansen.foo',
     'What you look like from the internet''s perspective - WAN IP, ISP, location, timezone and more. No server. No tracking. Pure client-side.',
     'globe', 1, 'echo.johansen.foo', 2),
    ('homelab', NULL,
     'Four-node Proxmox VE 9 cluster (dagobah, geonosis, mendavi, serenno) running Ceph storage, SDN, HA, and a mix of LXC containers and QEMU VMs. Because every good automation starts at home.',
     'server', 0, 'self-hosted // private', 3),
    ('icloud-mailflow', 'https://github.com/mojoaar/icloud-mailflow',
     'IMAP rules engine for iCloud Mail - AND/OR logic, 13 operators, auto-reply, webhooks, dry-run, and MCP server. Go + HTMX.',
     'mail', 1, 'github.com/mojoaar/icloud-mailflow', 4),
    ('ignite', 'https://ignite.johansen.foo',
     'Provisioning with a heartbeat - AI-guided conversational interviews that generate project specs, agent guides, implementation plans, and READMEs. Wails desktop GUI.',
     'flame', 1, 'ignite.johansen.foo', 5),
    ('kanzo', 'https://github.com/mojoaar/kanzo',
     'Kanban board with configurable columns, drag & drop, GitHub sync, people & category systems, 13 themes, and full PWA support.',
     'kanban', 1, 'github.com/mojoaar/kanzo', 6),
    ('krypt', 'https://krypt.johansen.foo',
     'Terminal password manager with AES-256-GCM encryption, Argon2id key derivation, optional 2FA, and GitHub Gist sync. Built with Bubble Tea.',
     'shield-check', 1, 'krypt.johansen.foo', 7),
    ('mindmatrix', 'https://mindmatrix.johansen.foo',
     'Markdown-first, self-hosted, multi-user knowledge hub - realtime collaboration, backlinks, version history, plugins, and full REST API.',
     'brain', 1, 'mindmatrix.johansen.foo', 8);

INSERT INTO experience (years, role, company, icon, sort) VALUES
    ('2022 – present', 'Team Manager, IT Server Operations',              'JYSK',                  'briefcase',       0),
    ('2020 – 2022',    'Team Leader, IT Server Operations Nordic',        'JYSK',                  'briefcase',       1),
    ('2018 – 2020',    'Senior Consultant, ServiceNow',                   'Devoteam',              'square-terminal', 2),
    ('2016 – 2018',    'Systems Consultant, ServiceNow & Azure',          'Syspeople ApS',         'square-terminal', 3),
    ('2015 – 2016',    'Automation Engineer',                             'Wolseley',              'square-terminal', 4),
    ('2013 – 2015',    'Enterprise Systems Management Administrator',     'Wolseley',              'square-terminal', 5),
    ('2007 – 2013',    'Senior Technical Analyst',                        'Wolseley',              'square-terminal', 6),
    ('1999 – 2007',    'Lance Corporal, Tank Squadron',                   'Jydske Dragonregiment', 'shield',          7);

INSERT INTO skill (name, sort) VALUES
    ('ITSM', 0), ('ESM', 1), ('ServiceNow', 2), ('Jira', 3), ('ITIL', 4),
    ('Azure', 5), ('Google Cloud', 6), ('VMware', 7), ('KVM', 8), ('Proxmox', 9),
    ('Terraform', 10), ('OpenTofu', 11), ('IaC', 12), ('Automation', 13),
    ('Scripting', 14), ('Python', 15), ('Go', 16), ('PowerShell', 17),
    ('Active Directory', 18), ('Entra ID', 19), ('Linux', 20), ('Windows', 21),
    ('MacOS', 22), ('AI', 23), ('Leadership', 24), ('Management', 25),
    ('Budget', 26), ('Hosting', 27), ('Web Development', 28), ('Presenting', 29);

INSERT INTO settings (key, value) VALUES
    ('posts_enabled',        'true'),
    ('active_theme',         'johansen'),
    ('stats_enabled',        'true'),
    ('stats_retention_days', '90'),
    ('timezone',             'Europe/Copenhagen'),
    ('site_title',           'Morten Johansen | johansen.foo'),
    ('title_template',       '%s | johansen.foo'),
    ('seo_description',      'Morten Johansen - Building and running complex infrastructure & cloud environments for 18+ years. Global ops leader, open-source tinkerer, and automation enthusiast.'),
    ('og_image_url',         'https://johansen.foo/static/avatar.png'),
    ('og_type',              'website'),
    ('twitter_card',         'summary'),
    ('canonical_base_url',   'https://johansen.foo'),
    ('noindex',              'false'),
    ('sitemap_enabled',      'true'),
    ('robots_txt',           'User-agent: *
Allow: /

Sitemap: https://johansen.foo/sitemap.xml');

INSERT INTO theme (slug, name, description, sort, created_at, updated_at)
VALUES (
    'johansen',
    'Johansen',
    'The original johansen.foo theme.',
    0,
    strftime('%Y-%m-%dT%H:%M:%SZ','now'),
    strftime('%Y-%m-%dT%H:%M:%SZ','now')
);
