import UIKit

class ViewController: UIViewController {

    override func viewDidLoad() {
        super.viewDidLoad()
        // Do any additional setup after loading the view.
        self.view.backgroundColor = .white
        
        let label = UILabel(frame: CGRect(x: 0, y: 0, width: 300, height: 50))
        label.center = self.view.center
        label.textAlignment = .center
        label.text = "MobileProxy Test App"
        self.view.addSubview(label)

        // In a real test app, you might trigger test logic here or observe its results.
        // For XCTests, this UI is mostly a host.
    }
}
