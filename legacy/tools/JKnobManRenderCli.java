import java.awt.AlphaComposite;
import java.awt.Color;
import java.awt.GraphicsEnvironment;
import java.awt.Graphics2D;
import java.io.File;
import java.lang.reflect.Field;
import java.io.IOException;
import java.util.Arrays;
import sun.misc.Unsafe;

public class JKnobManRenderCli
{
    public static void main(String[] args) throws Exception
    {
        Options opts = Options.parse(args);
        if (opts == null)
        {
            usage();
            System.exit(2);
            return;
        }

        installGUIEditorShim();
        Control ctl = new Control();

        int exitCode = 0;
        try
        {
            if (opts.samplesDir != null)
            {
                renderSamples(ctl, opts);
            }
            else if (opts.keyframes != null && opts.keyframes.length > 0)
            {
                renderKeyframes(ctl, opts.inputFile, opts.outputFile, opts);
            }
            else
            {
                renderOne(ctl, opts.inputFile, opts.outputFile, opts.frame, opts.useRenderFrames, opts.flattenBackground);
            }
        }
        catch (Exception ex)
        {
            ex.printStackTrace(System.err);
            exitCode = 1;
        }
        finally
        {
        }

        System.exit(exitCode);
    }

    private static void renderSamples(Control ctl, Options opts) throws IOException
    {
        File[] samples = opts.samplesDir.listFiles((dir, name) -> name.endsWith(".knob"));
        if (samples == null || samples.length == 0)
        {
            throw new IOException("no .knob files found in " + opts.samplesDir);
        }

        Arrays.sort(samples, (a, b) -> a.getName().compareToIgnoreCase(b.getName()));
        if (!opts.outputDir.exists() && !opts.outputDir.mkdirs())
        {
            throw new IOException("failed to create output directory " + opts.outputDir);
        }

        for (File sample : samples)
        {
            String name = sample.getName();
            int dot = name.lastIndexOf('.');
            if (dot > 0)
            {
                name = name.substring(0, dot);
            }

            if (!opts.wantsSample(name))
            {
                continue;
            }

            if (opts.keyframes != null && opts.keyframes.length > 0)
            {
                renderKeyframes(ctl, sample, opts.outputDir, name, opts);
                continue;
            }

            File out = new File(opts.outputDir, name + ".png");
            if (out.exists() && !opts.overwrite)
            {
                continue;
            }

            renderOne(ctl, sample, out, opts.frame, opts.useRenderFrames, opts.flattenBackground);
            System.out.println(out.getPath());
        }
    }

    private static void renderOne(
        Control ctl,
        File inputFile,
        File outputFile,
        int frame,
        boolean useRenderFrames,
        boolean flattenBackground
    )
        throws IOException
    {
        if (outputFile == null)
        {
            throw new IOException("missing output file");
        }

        Bitmap strip = renderStripForInput(ctl, inputFile, useRenderFrames, flattenBackground);
        Bitmap out = extractFrame(strip, ctl, frame);

        File parent = outputFile.getParentFile();
        if (parent != null && !parent.exists() && !parent.mkdirs())
        {
            throw new IOException("failed to create output directory " + parent);
        }

        out.Write(outputFile.getAbsolutePath(), "png");
    }

    private static void renderKeyframes(Control ctl, File inputFile, File outputFile, Options opts) throws IOException
    {
        Bitmap strip = renderStripForInput(ctl, inputFile, opts.useRenderFrames, opts.flattenBackground);
        for (String keyframe : opts.keyframes)
        {
            File out = keyframeOutputFile(outputFile, keyframe);
            Bitmap frame = extractFrame(strip, ctl, keyframeFrameIndex(keyframe, ctl.prefs.frames));
            File parent = out.getParentFile();
            if (parent != null && !parent.exists() && !parent.mkdirs())
            {
                throw new IOException("failed to create output directory " + parent);
            }
            frame.Write(out.getAbsolutePath(), "png");
        }
    }

    private static void renderKeyframes(Control ctl, File inputFile, File outputDir, String stem, Options opts) throws IOException
    {
        Bitmap strip = renderStripForInput(ctl, inputFile, opts.useRenderFrames, opts.flattenBackground);
        for (String keyframe : opts.keyframes)
        {
            File out = new File(outputDir, stem + "__" + keyframe + ".png");
            if (out.exists() && !opts.overwrite)
            {
                continue;
            }

            Bitmap frame = extractFrame(strip, ctl, keyframeFrameIndex(keyframe, ctl.prefs.frames));
            frame.Write(out.getAbsolutePath(), "png");
            System.out.println(out.getPath());
        }
    }

    private static Bitmap renderStripForInput(
        Control ctl,
        File inputFile,
        boolean useRenderFrames,
        boolean flattenBackground
    )
        throws IOException
    {
        if (inputFile == null)
        {
            throw new IOException("missing input file");
        }
        if (!inputFile.isFile())
        {
            throw new IOException("input file not found: " + inputFile);
        }
        loadKnob(ctl, inputFile, useRenderFrames);
        ctl.renderreq.WaitBreak();

        resolveExternalAssets(ctl);

        return renderStrip(ctl, flattenBackground);
    }

    private static void loadKnob(Control ctl, File inputFile, boolean useRenderFrames) throws IOException
    {
        ctl.render.Stop();
        ctl.renderreq.WaitBreak();
        ctl.iEdit = 0;
        ctl.strCurrentFile = inputFile.getAbsolutePath();
        ctl.strKnobDir = Pathname.GetDir(ctl.strCurrentFile);
        ctl.iCurrentLayer = 1;

        ProfileReader ppr = new ProfileReader(ctl.strCurrentFile);
        if (ppr.Error() != 0)
        {
            throw new IOException("failed to open " + ctl.strCurrentFile);
        }

        ppr.SetSection("Prefs");
        ctl.prefs.rendermode = useRenderFrames ? 1 : 0;
        ctl.prefs.pwidth.val = ppr.ReadInt("OutputSizeX", 32);
        ctl.prefs.pheight.val = ppr.ReadInt("OutputSizeY", 32);
        ctl.prefs.oversampling.val = ppr.ReadInt("OverSampling", 0);
        ctl.prefs.alignhorz.val = ppr.ReadInt("AlignHorizontal", 0);
        ctl.prefs.priFramesPrev.val = ctl.prefs.priFramesRender.val = ppr.ReadInt("NumOfImage", 0);
        if (ctl.prefs.priFramesPrev.val <= 0)
        {
            ctl.prefs.priFramesRender.val = ppr.ReadInt("RenderFrames", 31);
            ctl.prefs.priFramesPrev.val = ppr.ReadInt("PreviewFrames", 5);
        }

        ctl.layers.clear();
        ctl.iMaxLayer = ppr.ReadInt("Layers", 1);
        for (int i = 0; i < ctl.iMaxLayer; ++i)
        {
            ctl.layers.add(new Layer(ctl, true));
        }

        setRenderSize(ctl);
        ctl.prefs.bkcolor.col.SetRgb(ppr.ReadInt("BkColorR", 255), ppr.ReadInt("BkColorG", 255), ppr.ReadInt("BkColorB", 255));

        for (int i = 0; i < 8; ++i)
        {
            int ii = i + 1;
            ctl.animcurve[i].lv[0] = ppr.ReadInt("Curve" + ii + "L0", 0);
            for (int j = 1; j < 11; ++j)
            {
                String n = (new String[] {"0", "1", "2", "3", "4", "a", "b", "c", "d", "e", "f", "5"})[j];
                ctl.animcurve[i].tm[j] = ppr.ReadInt("Curve" + ii + "T" + n, -1);
                ctl.animcurve[i].lv[j] = ppr.ReadInt("Curve" + ii + "L" + n, -1);
            }
            ctl.animcurve[i].lv[11] = ppr.ReadInt("Curve" + ii + "L5", 100);
            ctl.animcurve[i].tm[0] = 0;
            ctl.animcurve[i].tm[11] = 100;
            ctl.animcurve[i].stepreso.val = ppr.ReadInt("Curve" + ii + "StepReso", 0);
        }

        for (int i = 0; i < ctl.layers.size(); ++i)
        {
            ctl.layers.get(i).pcVisible.val = ppr.ReadInt("Visible1_" + i, -1);
        }

        ppr.SetSection("Pal");
        for (int i = 0; i < 24; ++i)
        {
            int c = ppr.ReadInt("Pal" + i, -1);
            if (c >= 0)
            {
                ctl.pal[i] = c;
            }
        }

        for (int i = 0; i < ctl.layers.size(); ++i)
        {
            ctl.LoadLayer(ppr, ctl.layers.get(i), i);
        }
    }

    private static void setRenderSize(Control ctl)
    {
        ctl.prefs.width = ctl.prefs.pwidth.val * (1 << ctl.prefs.oversampling.val);
        ctl.prefs.height = ctl.prefs.pheight.val * (1 << ctl.prefs.oversampling.val);
        ctl.prefs.frames = ctl.prefs.rendermode == 0 ? ctl.prefs.priFramesPrev.val : ctl.prefs.priFramesRender.val;
        if (ctl.prefs.testindex >= ctl.prefs.priFramesRender.val)
        {
            ctl.prefs.testindex = ctl.prefs.priFramesRender.val - 1;
        }

        for (int i = 0; i < ctl.iMaxLayer; ++i)
        {
            Layer ly = ctl.layers.get(i);
            ly.imgprevf = new Bitmap(ctl.prefs.width, ctl.prefs.height);
            ly.imgprevt = new Bitmap(ctl.prefs.width, ctl.prefs.height);
            ly.prim.SetSize(ctl.prefs);
        }

        int stripw;
        int striph;
        if (ctl.prefs.alignhorz.val == 0)
        {
            stripw = ctl.prefs.width;
            striph = ctl.prefs.height * ctl.prefs.frames;
        }
        else
        {
            stripw = ctl.prefs.width * ctl.prefs.frames;
            striph = ctl.prefs.height;
        }
        ctl.imgRender = new Bitmap(stripw, striph);
    }

    private static void resolveExternalAssets(Control ctl)
    {
        for (int i = 0; i < ctl.layers.size(); ++i)
        {
            Layer ly = ctl.layers.get(i);
            if (ly.prim.texturedepth.val != 0.0 && ly.prim.texturename != null && ly.prim.texturename.length() > 0 &&
                ly.prim.tex == null)
            {
                ly.prim.tex = ly.prim.tex0 = resolveTexture(ctl, ly, i);
            }

            if (ly.prim.type.val == 1 && ly.prim.bmpImage == null && ly.prim.file.val != null && ly.prim.file.val.length() > 0)
            {
                File image = resolveFile(ctl.strKnobDir, ly.prim.file.val);
                if (image != null && image.isFile())
                {
                    ly.LoadImage(image.getAbsolutePath());
                }
            }
        }
    }

    private static Tex resolveTexture(Control ctl, Layer ly, int index)
    {
        for (int i = 0; i < index; ++i)
        {
            Layer prev = ctl.layers.get(i);
            if (prev.prim.tex != null && ly.prim.texturename.equals(prev.prim.texturename))
            {
                return prev.prim.tex;
            }
        }

        if (ctl.fileTextures == null)
        {
            return null;
        }

        String want = ly.prim.texturename;
        for (File file : ctl.fileTextures)
        {
            String name = file.getName();
            if (name.equalsIgnoreCase(want) || stripExt(name).equalsIgnoreCase(want))
            {
                return new Tex(file.getAbsolutePath());
            }
        }

        File local = resolveFile(ctl.strKnobDir, want);
        if (local != null && local.isFile())
        {
            return new Tex(local.getAbsolutePath());
        }

        return null;
    }

    private static File resolveFile(String baseDir, String path)
    {
        File file = new File(path);
        if (file.isAbsolute())
        {
            return file;
        }
        if (baseDir == null || baseDir.length() == 0)
        {
            return file;
        }
        return new File(baseDir, path);
    }

    private static String stripExt(String name)
    {
        int dot = name.lastIndexOf('.');
        if (dot <= 0)
        {
            return name;
        }
        return name.substring(0, dot);
    }

    private static Bitmap renderStrip(Control ctl, boolean flattenBackground)
    {
        applyLayerVisibility(ctl);
        ctl.imgRender.Clear(new Color(0, 0, 0, 0));
        ctl.bHasAlpha = !flattenBackground;

        for (int frame = 0; frame < ctl.prefs.frames; ++frame)
        {
            int px = ctl.prefs.alignhorz.val != 0 ? ctl.prefs.width * frame : 0;
            int py = ctl.prefs.alignhorz.val != 0 ? 0 : ctl.prefs.height * frame;
            ctl.imgRender.ClearRect(px, py, ctl.prefs.width, ctl.prefs.height, new Color(0, 0, 0, 0));

            for (int layer = 0; layer < ctl.layers.size(); ++layer)
            {
                Layer ly = ctl.layers.get(layer);
                if (ly.visible != 0)
                {
                    ly.RenderFrame(ctl.imgRender, true, px, py, ctl.prefs.width, ctl.prefs.height, frame, ctl.prefs.frames - 1, true);
                }
            }
        }

        Bitmap bmp;
        if (ctl.prefs.oversampling.val != 0)
        {
            int w = ctl.imgRender.width >> ctl.prefs.oversampling.val;
            int h = ctl.imgRender.height >> ctl.prefs.oversampling.val;
            bmp = new Bitmap(w, h);
            ctl.imgRender.DecimationTo(bmp, 0, 0, w, h, 0, 0, ctl.imgRender.width, ctl.imgRender.height);
        }
        else
        {
            bmp = ctl.imgRender;
        }

        if (!ctl.bHasAlpha)
        {
            Graphics2D g2 = (Graphics2D)bmp.img.getGraphics();
            g2.setComposite(AlphaComposite.DstOver);
            g2.setColor(new Color(ctl.prefs.bkcolor.col.r, ctl.prefs.bkcolor.col.g, ctl.prefs.bkcolor.col.b));
            g2.fillRect(0, 0, bmp.width, bmp.height);
        }

        return bmp;
    }

    private static void applyLayerVisibility(Control ctl)
    {
        int soloIndex = -1;
        for (int i = 0; i < ctl.iMaxLayer; ++i)
        {
            Layer ly = ctl.layers.get(i);
            if (ly.pcSolo.val != 0)
            {
                soloIndex = i;
                break;
            }
        }

        for (int i = 0; i < ctl.iMaxLayer; ++i)
        {
            Layer ly = ctl.layers.get(i);
            ly.visible = soloIndex >= 0 ? (ly.pcSolo.val != 0 ? 1 : 0) : (ly.pcVisible.val != 0 ? 1 : 0);
        }
    }

    private static Bitmap extractFrame(Bitmap strip, Control ctl, int frame)
    {
        int frames = Math.max(1, ctl.prefs.frames);
        int index = frame;
        if (index < 0)
        {
            index = 0;
        }
        if (index >= frames)
        {
            index = frames - 1;
        }

        int frameW = ctl.prefs.pwidth.val;
        int frameH = ctl.prefs.pheight.val;
        Bitmap out = new Bitmap(frameW, frameH);
        Graphics2D g2 = (Graphics2D)out.img.getGraphics();
        g2.setComposite(AlphaComposite.Src);

        int sx = ctl.prefs.alignhorz.val != 0 ? index * frameW : 0;
        int sy = ctl.prefs.alignhorz.val != 0 ? 0 : index * frameH;
        g2.drawImage(strip.img, 0, 0, frameW, frameH, sx, sy, sx + frameW, sy + frameH, null);
        return out;
    }

    private static void installGUIEditorShim() throws Exception
    {
        Field instField = GUIEditor.class.getDeclaredField("inst");
        instField.setAccessible(true);
        if (instField.get(null) != null)
        {
            return;
        }

        Unsafe unsafe = getUnsafe();
        GUIEditor shim = (GUIEditor)unsafe.allocateInstance(GUIEditor.class);
        shim.fonts = GraphicsEnvironment.getLocalGraphicsEnvironment().getAvailableFontFamilyNames();
        shim.bmpNone = new Bitmap(16, 16);
        instField.set(null, shim);
    }

    private static Unsafe getUnsafe() throws Exception
    {
        Field f = Unsafe.class.getDeclaredField("theUnsafe");
        f.setAccessible(true);
        return (Unsafe)f.get(null);
    }

    private static void usage()
    {
        System.err.println("Usage:");
        System.err.println("  JKnobManRenderCli --input <file.knob> --output <file.png> [--frame <n>] [--preview-frames] [--flatten-bg]");
        System.err.println(
            "  JKnobManRenderCli --input <file.knob> --output <file.png> [--keyframes first,mid,last] [--preview-frames] [--flatten-bg]"
        );
        System.err.println(
            "  JKnobManRenderCli --samples <dir> --output-dir <dir> [--frame <n>] [--keyframes first,mid,last] [--names a,b] [--overwrite] [--preview-frames] [--flatten-bg]"
        );
    }

    private static File keyframeOutputFile(File baseOutput, String keyframe)
    {
        String name = baseOutput.getName();
        int dot = name.lastIndexOf('.');
        String renderedName = dot >= 0 ? name.substring(0, dot) + "__" + keyframe + name.substring(dot) : name + "__" + keyframe + ".png";
        File parent = baseOutput.getParentFile();
        return parent == null ? new File(renderedName) : new File(parent, renderedName);
    }

    private static int keyframeFrameIndex(String keyframe, int totalFrames)
    {
        if (totalFrames <= 1)
        {
            return 0;
        }

        switch (keyframe)
        {
            case "first":
                return 0;
            case "mid":
                return totalFrames / 2;
            case "last":
                return totalFrames - 1;
            default:
                throw new IllegalArgumentException("unsupported keyframe: " + keyframe);
        }
    }

    private static final class Options
    {
        File inputFile;
        File outputFile;
        File samplesDir;
        File outputDir;
        String[] names;
        String[] keyframes;
        int frame = 0;
        boolean overwrite;
        boolean useRenderFrames = true;
        boolean flattenBackground;

        static Options parse(String[] args)
        {
            Options opts = new Options();
            for (int i = 0; i < args.length; ++i)
            {
                String arg = args[i];
                if ("--input".equals(arg) && i + 1 < args.length)
                {
                    opts.inputFile = new File(args[++i]);
                }
                else if ("--output".equals(arg) && i + 1 < args.length)
                {
                    opts.outputFile = new File(args[++i]);
                }
                else if ("--samples".equals(arg) && i + 1 < args.length)
                {
                    opts.samplesDir = new File(args[++i]);
                }
                else if ("--output-dir".equals(arg) && i + 1 < args.length)
                {
                    opts.outputDir = new File(args[++i]);
                }
                else if ("--names".equals(arg) && i + 1 < args.length)
                {
                    opts.names = splitList(args[++i]);
                }
                else if ("--keyframes".equals(arg) && i + 1 < args.length)
                {
                    opts.keyframes = splitList(args[++i]);
                }
                else if ("--frame".equals(arg) && i + 1 < args.length)
                {
                    opts.frame = Integer.parseInt(args[++i]);
                }
                else if ("--overwrite".equals(arg))
                {
                    opts.overwrite = true;
                }
                else if ("--preview-frames".equals(arg))
                {
                    opts.useRenderFrames = false;
                }
                else if ("--flatten-bg".equals(arg))
                {
                    opts.flattenBackground = true;
                }
                else
                {
                    return null;
                }
            }

            if (opts.samplesDir != null)
            {
                if (opts.outputDir == null)
                {
                    return null;
                }
                return opts;
            }

            if (opts.inputFile == null || opts.outputFile == null)
            {
                return null;
            }
            return opts;
        }

        boolean wantsSample(String name)
        {
            if (this.names == null || this.names.length == 0)
            {
                return true;
            }

            for (String candidate : this.names)
            {
                if (candidate.equals(name))
                {
                    return true;
                }
            }

            return false;
        }

        static String[] splitList(String raw)
        {
            if (raw == null || raw.trim().length() == 0)
            {
                return new String[0];
            }

            return Arrays.stream(raw.split(","))
                .map(String::trim)
                .filter(s -> s.length() > 0)
                .toArray(String[]::new);
        }
    }
}
