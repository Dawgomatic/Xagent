# User Basic Profile Query Schema

Read from profile.json and visualize user basic medical parameters.

## Read Data

Use `Read` tool to read `data/profile.json`.

## Display Fields

### Basic Information

| Field | Description |
|-------|-------------|
| Gender | M=Male, F=Female |
| Height | cm |
| Weight | kg |
| Birth Date | YYYY-MM-DD |
| Age | Auto-calculated |

### Health Indicators

| Field | Description |
|-------|-------------|
| BMI | Weight/Height² |
| BMI Status | Underweight/Normal/Overweight/Obese |
| Body Surface Area | For radiation dose correction |

## Visual Elements

- BMI index visualization bar
- Weight trend chart
- Status icons (Normal Warning Danger)

## Data Storage

- Read location: `data/profile.json`
