# ComfyUI-OpenClaw Skill 

Skill สำหรับเชื่อมต่อและสั่งงาน ComfyUI API อย่างมีประสิทธิภาพ (ประหยัด Token สูงสุด) โดยใช้ระบบ Template Mapping และ Auto-Asset Management.

##  โครงสร้างของ Skill
- **Host Address:** `192.168.1.38:8190` (บันทึกไว้ใน TOOLS.md)
- **Workflow Directory:** `skills/comfyui/workflows/` (อยู่ภายใต้โฟลเดอร์ของ Skill)
- **Output Directory:** `outputs/comfy/` (Relative to workspace root)
- **Core Script:** `skills/comfyui/comfy_client.py`

##  เครื่องมือที่ใช้งาน (CLI)
เรียกใช้ผ่าน `exec` command:
`python3 skills/comfyui/comfy_client.py <template_id> "<prompt>" [input_image_path/orientation] [orientation]`

- **orientation:** ระบุ `portrait` (720x1280) หรือ `landscape` (1280x720). ค่าเริ่มต้นคือ `portrait`.

### ตัวเลือก Template ที่มีตอนนี้:
1. `gen_z`: สำหรับสร้างรูปใหม่ (image_z_image_turbo.json)
2. `qwen_edit`: สำหรับแก้ไขรูปภาพ (qwen_image_edit_2511.json) - **รองรับ Auto-Upload**

##  วิธีการเพิ่ม Workflow ใหม่ (Scalability)
หากคุณ Salmon มี workflow ใหม่ (.json) สามารถใช้งานได้โดย:
1. นำไฟล์ JSON ไปวางไว้ที่ `skills/comfyui/workflows/`
2. อัปเดตตัวแปร `WORKFLOW_MAP` ในไฟล์ `skills/comfyui/comfy_client.py` เพิ่ม ID และ Path ของไฟล์ใหม่
3. (Optional) หาก Workflow ใช้ Node เฉพาะเจาะจง ให้ปรับ Logic การ Inject ในส่วน `main()` ของสคริปต์

##  กลยุทธ์ประหยัด Token และทรัพยากร
- **No JSON in Prompt:** ห้ามส่งโครงสร้าง Workflow ลงในแชท ให้ใช้ `template_id` แทน
- **Path-Based Messaging:** อ้างอิงไฟล์ภาพด้วย Local Path (Relative) แทนการใช้ Base64 ใน Prompt
- **Direct Delivery:** ส่งภาพให้ผู้ใช้ผ่าน Telegram หรือเปิดบนเครื่อง Mac โดยตรงเพื่อลดภาระ Context Window ของ LLM
