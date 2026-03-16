-- name: LinkModifierGroupToMenuItem :exec
INSERT INTO menu_item_modifiers (menu_item_id, modifier_group_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: DeleteMenuItemModifiers :exec
DELETE FROM menu_item_modifiers
WHERE menu_item_id = $1;

-- name: GetModifierGroupIDsByMenuItem :many
SELECT modifier_group_id FROM menu_item_modifiers
WHERE menu_item_id = $1;

-- name: GetModifierGroupsWithOptionsByMenuItem :many
SELECT
    mg.id           AS group_id,
    mg.name         AS group_name,
    mg.is_required,
    mg.max_select,
    mo.id           AS option_id,
    mo.name         AS option_name,
    mo.extra_price
FROM menu_item_modifiers mim
JOIN modifier_groups mg ON mg.id = mim.modifier_group_id
JOIN modifier_options mo ON mo.group_id = mg.id
WHERE mim.menu_item_id = $1
ORDER BY mg.name, mo.name;
