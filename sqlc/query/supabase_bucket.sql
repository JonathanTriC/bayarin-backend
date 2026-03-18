-- Create public bucket for menu images
INSERT INTO storage.buckets (id, name, public)
VALUES ('menu-images', 'menu-images', true)
ON CONFLICT DO NOTHING;

-- Allow service role to upload
CREATE POLICY "Service role can upload menu images"
ON storage.objects FOR INSERT TO service_role
WITH CHECK (bucket_id = 'menu-images');

-- Allow public read (bucket is public)
CREATE POLICY "Public can view menu images"
ON storage.objects FOR SELECT TO public
USING (bucket_id = 'menu-images');

-- Allow service role to delete (for image replacement)
CREATE POLICY "Service role can delete menu images"
ON storage.objects FOR DELETE TO service_role
USING (bucket_id = 'menu-images');
