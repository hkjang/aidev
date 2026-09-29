import org.apache.poi.xssf.streaming.SXSSFWorkbook;
import org.apache.poi.ss.util.WorkbookUtil;
class PoiSheetProbe {
 public static void main(String[] args) throws Exception {
  for(String id : new String[]{"list_export_ok", "sales:2026", "sales_abcdefghijklmnopqrstuvwxyz_123456789"}) {
   try (var wb=new SXSSFWorkbook(100)) {
    try { System.out.println(id+" => "+wb.createSheet(id).getSheetName()); }
    catch(IllegalArgumentException e) {System.out.println(id+" => "+e.getClass().getSimpleName()+": "+e.getMessage());}
   }
   try (var wb=new SXSSFWorkbook(100)) {
    System.out.println("safe => "+wb.createSheet(WorkbookUtil.createSafeSheetName(id)).getSheetName());
   }
  }
 }
}
